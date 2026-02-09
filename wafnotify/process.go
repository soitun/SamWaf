package wafnotify

import (
	"SamWaf/wafnotify/kafka"
	"SamWaf/wafnotify/logfilewriter"
	"fmt"
	"strings"
)

func InitNotifyKafkaEngine(enable int64, url string, topic string) *WafNotifyService {

	brokers := strings.Split(url, ",") // Kafka brokers
	notifier, err := kafka.NewKafkaNotifier(brokers, topic, enable)
	if err != nil {
		fmt.Printf("Failed to create notifier: %v\n", err)
		return NewWafNotifyService(notifier, enable)
	}
	// 创建日志服务并注入 notifier
	return NewWafNotifyService(notifier, enable)
}

// InitLogFileWriterEngine 初始化日志文件写入引擎
func InitLogFileWriterEngine(enable int64, filePath, format, customTpl string, maxSize int64, maxBackups int, maxDays int, compress bool) *WafNotifyService {
	notifier, err := logfilewriter.NewLogFileWriter(filePath, format, customTpl, maxSize, maxBackups, maxDays, compress)
	if err != nil {
		fmt.Printf("Failed to create log file writer: %v\n", err)
		return NewWafNotifyService(notifier, enable)
	}
	return NewWafNotifyService(notifier, enable)
}
