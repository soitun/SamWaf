package wafqueue

import (
	"SamWaf/common/zlog"
	"SamWaf/global"
	"SamWaf/service/waf_service"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	notifyAggMaxBuffer     = 100              // 聚合缓冲区最大条目数，超过立即刷新
	notifyAggMaxDetail     = 10               // 合并通知中最多展示的详细条目数
	notifyAggFlushInterval = 10 * time.Second // 聚合刷新间隔（默认10秒）
)

// notifyEntry 通知聚合条目
type notifyEntry struct {
	MessageType string // 消息类型（用于按类型分组和路由到对应订阅渠道）
	Title       string // 通知标题
	Content     string // 通知正文
}

// notifyAggregator 通知聚合器，在时间窗口内按消息类型分组合并通知，避免通知轰炸
type notifyAggregator struct {
	mu     sync.Mutex
	buffer []notifyEntry
}

// gNotifyAgg 全局通知聚合器实例
var gNotifyAgg = &notifyAggregator{
	buffer: make([]notifyEntry, 0),
}

// Add 添加一条通知到聚合缓冲区，超过上限自动刷新
func (a *notifyAggregator) Add(entry notifyEntry) {
	a.mu.Lock()
	a.buffer = append(a.buffer, entry)
	shouldFlush := len(a.buffer) >= notifyAggMaxBuffer
	a.mu.Unlock()

	if shouldFlush {
		a.Flush()
	}
}

// Flush 刷新缓冲区，按消息类型分组合并后发送
func (a *notifyAggregator) Flush() {
	defer func() {
		if r := recover(); r != nil {
			zlog.Error(fmt.Sprintf("通知聚合器Flush发生panic: %v", r))
		}
	}()

	a.mu.Lock()
	if len(a.buffer) == 0 {
		a.mu.Unlock()
		return
	}
	entries := a.buffer
	a.buffer = make([]notifyEntry, 0)
	a.mu.Unlock()

	// 按消息类型分组（保证不同类型走各自的订阅渠道）
	groups := make(map[string][]notifyEntry)
	for _, e := range entries {
		groups[e.MessageType] = append(groups[e.MessageType], e)
	}

	// 逐组发送
	totalSent := 0
	for msgType, groupEntries := range groups {
		if len(groupEntries) == 1 {
			// 单条直接发送（保持原有格式不变）
			e := groupEntries[0]
			waf_service.WafNotifySenderServiceApp.SendNotification(msgType, e.Title, e.Content)
		} else {
			// 多条合并为一条摘要通知
			title := fmt.Sprintf("%s（合并%d条）", groupEntries[0].Title, len(groupEntries))

			showCount := len(groupEntries)
			if showCount > notifyAggMaxDetail {
				showCount = notifyAggMaxDetail
			}

			var parts []string
			for i := 0; i < showCount; i++ {
				parts = append(parts, fmt.Sprintf("**[%d]** %s", i+1, groupEntries[i].Content))
			}
			if len(groupEntries) > notifyAggMaxDetail {
				parts = append(parts,
					fmt.Sprintf("\n...及其他 %d 条记录", len(groupEntries)-notifyAggMaxDetail))
			}

			content := strings.Join(parts, "\n\n---\n\n")
			waf_service.WafNotifySenderServiceApp.SendNotification(msgType, title, content)
		}
		totalSent += len(groupEntries)
	}

	zlog.Debug("通知聚合器刷新，共发送" + strconv.Itoa(totalSent) + "条通知（" + strconv.Itoa(len(groups)) + "个分组）")
}

// StartFlushLoop 启动定时刷新协程，收到关闭信号时执行最终刷新
func (a *notifyAggregator) StartFlushLoop() {
	defer func() {
		if r := recover(); r != nil {
			zlog.Error(fmt.Sprintf("通知聚合器FlushLoop发生panic: %v，将在3秒后自动重启", r))
			time.Sleep(3 * time.Second)
			go a.StartFlushLoop() // panic后自动重启，确保聚合器不会永久停止
		}
	}()

	ticker := time.NewTicker(notifyAggFlushInterval)
	defer ticker.Stop()
	for {
		select {
		case <-global.GWAF_QUEUE_SHUTDOWN_SIGNAL:
			a.Flush() // 关闭前最终刷新，确保不丢失缓冲中的通知
			zlog.Info("通知聚合器收到关闭信号，已完成最终刷新")
			return
		case <-ticker.C:
			a.Flush()
		}
	}
}
