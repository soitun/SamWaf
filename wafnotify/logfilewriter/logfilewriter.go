package logfilewriter

import (
	"SamWaf/common/zlog"
	"SamWaf/innerbean"
	"bufio"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

// LogFileWriter 实现 WafNotify 接口，将日志写入文件
type LogFileWriter struct {
	mu          sync.Mutex
	filePath    string // 日志文件路径
	format      string // 日志格式: nginx, apache, custom
	customTpl   string // 自定义格式模板
	maxSize     int64  // 单文件最大大小 (MB)
	maxBackups  int    // 保留的历史文件数量
	maxDays     int    // 保留天数
	compress    bool   // 是否压缩历史文件
	file        *os.File
	writer      *bufio.Writer
	currentSize int64  // 当前文件大小
	template    string // 解析后的模板
}

// NewLogFileWriter 创建日志文件写入器
func NewLogFileWriter(filePath, format, customTpl string, maxSize int64, maxBackups int, maxDays int, compress bool) (*LogFileWriter, error) {
	lw := &LogFileWriter{
		filePath:   filePath,
		format:     format,
		customTpl:  customTpl,
		maxSize:    maxSize,
		maxBackups: maxBackups,
		maxDays:    maxDays,
		compress:   compress,
	}

	// 解析模板
	lw.resolveTemplate()

	// 打开文件
	if err := lw.openFile(); err != nil {
		return lw, err
	}

	return lw, nil
}

// resolveTemplate 解析格式模板
func (lw *LogFileWriter) resolveTemplate() {
	if lw.format == "custom" && lw.customTpl != "" {
		lw.template = lw.customTpl
	} else {
		lw.template = GetFormatTemplate(lw.format)
	}
}

// openFile 打开或创建日志文件
func (lw *LogFileWriter) openFile() error {
	// 确保目录存在
	dir := filepath.Dir(lw.filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("创建日志目录失败: %v", err)
	}

	f, err := os.OpenFile(lw.filePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("打开日志文件失败: %v", err)
	}

	// 获取当前文件大小
	info, err := f.Stat()
	if err != nil {
		f.Close()
		return fmt.Errorf("获取文件信息失败: %v", err)
	}

	lw.file = f
	lw.writer = bufio.NewWriterSize(f, 64*1024) // 64KB缓冲区
	lw.currentSize = info.Size()

	return nil
}

// NotifySingle 实现 WafNotify 接口 - 写入单条日志
func (lw *LogFileWriter) NotifySingle(log *innerbean.WebLog) error {
	lw.mu.Lock()
	defer lw.mu.Unlock()

	if lw.file == nil {
		if err := lw.openFile(); err != nil {
			return err
		}
	}

	line := FormatLog(log, lw.template) + "\n"
	n, err := lw.writer.WriteString(line)
	if err != nil {
		return fmt.Errorf("写入日志失败: %v", err)
	}
	lw.currentSize += int64(n)

	// 检查是否需要轮转
	if lw.needsRotation() {
		if err := lw.rotate(); err != nil {
			zlog.Error("日志文件轮转失败: " + err.Error())
		}
	}

	return nil
}

// NotifyBatch 实现 WafNotify 接口 - 批量写入日志
func (lw *LogFileWriter) NotifyBatch(logs []*innerbean.WebLog) error {
	lw.mu.Lock()
	defer lw.mu.Unlock()

	if lw.file == nil {
		if err := lw.openFile(); err != nil {
			return err
		}
	}

	for _, log := range logs {
		line := FormatLog(log, lw.template) + "\n"
		n, err := lw.writer.WriteString(line)
		if err != nil {
			return fmt.Errorf("写入日志失败: %v", err)
		}
		lw.currentSize += int64(n)

		// 每条写入后检查是否需要轮转，避免单批数据过大时文件超限过多
		if lw.needsRotation() {
			// 先刷新再轮转
			if err := lw.writer.Flush(); err != nil {
				zlog.Error("刷新日志缓冲区失败: " + err.Error())
			}
			if err := lw.rotate(); err != nil {
				zlog.Error("日志文件轮转失败: " + err.Error())
			}
		}
	}

	// 刷新缓冲区
	if lw.writer != nil {
		if err := lw.writer.Flush(); err != nil {
			zlog.Error("刷新日志缓冲区失败: " + err.Error())
		}
	}

	return nil
}

// needsRotation 检查是否需要轮转
func (lw *LogFileWriter) needsRotation() bool {
	maxBytes := lw.maxSize * 1024 * 1024
	return lw.currentSize >= maxBytes
}

// rotate 执行日志文件轮转
func (lw *LogFileWriter) rotate() error {
	// 刷新并关闭当前文件
	if lw.writer != nil {
		lw.writer.Flush()
	}
	if lw.file != nil {
		lw.file.Close()
		lw.file = nil
		lw.writer = nil
	}

	// 轮转文件: access.log -> access.log.1.log -> access.log.2.log ...
	// 先移动已有的备份文件
	for i := lw.maxBackups; i >= 1; i-- {
		src := lw.backupName(i)
		dst := lw.backupName(i + 1)
		if _, err := os.Stat(src); err == nil {
			if i == lw.maxBackups {
				// 最大编号的文件直接删除
				os.Remove(src)
			} else {
				os.Rename(src, dst)
			}
		}
		// 同时处理压缩文件
		srcGz := src + ".gz"
		dstGz := dst + ".gz"
		if _, err := os.Stat(srcGz); err == nil {
			if i == lw.maxBackups {
				os.Remove(srcGz)
			} else {
				os.Rename(srcGz, dstGz)
			}
		}
	}

	// 当前文件重命名为 .1.log
	backup1 := lw.backupName(1)
	if err := os.Rename(lw.filePath, backup1); err != nil {
		zlog.Error("重命名日志文件失败: " + err.Error())
	}

	// 压缩刚轮转的文件
	if lw.compress {
		go lw.compressFile(backup1)
	}

	// 清理过期文件
	go lw.cleanOldFiles()

	// 打开新文件
	return lw.openFile()
}

// backupName 生成备份文件名
// 例如: logs/access.log -> logs/access.log.1.log
func (lw *LogFileWriter) backupName(index int) string {
	return lw.filePath + "." + strconv.Itoa(index) + ".log"
}

// compressFile 压缩文件
func (lw *LogFileWriter) compressFile(srcPath string) {
	src, err := os.Open(srcPath)
	if err != nil {
		zlog.Error("打开待压缩文件失败: " + err.Error())
		return
	}
	defer src.Close()

	dstPath := srcPath + ".gz"
	dst, err := os.Create(dstPath)
	if err != nil {
		zlog.Error("创建压缩文件失败: " + err.Error())
		return
	}
	defer dst.Close()

	gz := gzip.NewWriter(dst)
	defer gz.Close()

	if _, err := io.Copy(gz, src); err != nil {
		zlog.Error("压缩文件失败: " + err.Error())
		os.Remove(dstPath)
		return
	}

	gz.Close()
	dst.Close()
	src.Close()

	// 删除原始文件
	os.Remove(srcPath)
}

// cleanOldFiles 清理超期文件
func (lw *LogFileWriter) cleanOldFiles() {
	dir := filepath.Dir(lw.filePath)
	baseName := filepath.Base(lw.filePath)

	entries, err := os.ReadDir(dir)
	if err != nil {
		zlog.Error("读取日志目录失败: " + err.Error())
		return
	}

	now := time.Now()
	maxAge := time.Duration(lw.maxDays) * 24 * time.Hour

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		// 匹配备份文件: baseName.N.log 或 baseName.N.log.gz
		if !strings.HasPrefix(name, baseName+".") {
			continue
		}
		// 跳过当前日志文件本身
		if name == baseName {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			continue
		}

		// 按天数清理
		if lw.maxDays > 0 && now.Sub(info.ModTime()) > maxAge {
			fullPath := filepath.Join(dir, name)
			os.Remove(fullPath)
			zlog.Info("清理过期日志文件: " + fullPath)
		}
	}
}

// Close 关闭日志文件
func (lw *LogFileWriter) Close() {
	lw.mu.Lock()
	defer lw.mu.Unlock()

	if lw.writer != nil {
		lw.writer.Flush()
	}
	if lw.file != nil {
		lw.file.Close()
		lw.file = nil
		lw.writer = nil
	}
}

// GetLogPreview 获取日志文件最新N行（供前端预览）
func (lw *LogFileWriter) GetLogPreview(lines int) ([]string, error) {
	lw.mu.Lock()
	// 先刷新缓冲区以确保读到最新内容
	if lw.writer != nil {
		lw.writer.Flush()
	}
	lw.mu.Unlock()

	f, err := os.Open(lw.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, err
	}
	defer f.Close()

	return readLastLines(f, lines)
}

// readLastLines 读取文件最后N行
func readLastLines(f *os.File, n int) ([]string, error) {
	info, err := f.Stat()
	if err != nil {
		return nil, err
	}

	fileSize := info.Size()
	if fileSize == 0 {
		return []string{}, nil
	}

	// 从文件末尾向前读取
	bufSize := int64(8192)
	if bufSize > fileSize {
		bufSize = fileSize
	}

	var allLines []string
	offset := fileSize
	remaining := ""

	for offset > 0 {
		readSize := bufSize
		if readSize > offset {
			readSize = offset
		}
		offset -= readSize

		buf := make([]byte, readSize)
		_, err := f.ReadAt(buf, offset)
		if err != nil && err != io.EOF {
			return nil, err
		}

		chunk := string(buf) + remaining
		lines := strings.Split(chunk, "\n")

		// 第一段可能不完整，保留到下次
		remaining = lines[0]
		lines = lines[1:]

		// 逆序添加
		for i := len(lines) - 1; i >= 0; i-- {
			line := strings.TrimRight(lines[i], "\r")
			if line != "" {
				allLines = append([]string{line}, allLines...)
			}
		}

		if len(allLines) >= n {
			break
		}
	}

	// 处理最后剩余的部分
	if remaining != "" && len(allLines) < n {
		remaining = strings.TrimRight(remaining, "\r")
		if remaining != "" {
			allLines = append([]string{remaining}, allLines...)
		}
	}

	// 只返回最后N行
	if len(allLines) > n {
		allLines = allLines[len(allLines)-n:]
	}

	return allLines, nil
}

// BackupFileInfo 备份文件信息
type BackupFileInfo struct {
	Name     string `json:"name"`      // 文件名
	Size     int64  `json:"size"`      // 文件大小 (bytes)
	ModTime  string `json:"mod_time"`  // 修改时间
	FullPath string `json:"full_path"` // 完整路径
}

// GetBackupFiles 获取备份文件列表
func (lw *LogFileWriter) GetBackupFiles() ([]BackupFileInfo, error) {
	dir := filepath.Dir(lw.filePath)
	baseName := filepath.Base(lw.filePath)

	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return []BackupFileInfo{}, nil
		}
		return nil, err
	}

	var files []BackupFileInfo
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		// 匹配备份文件
		if !strings.HasPrefix(name, baseName+".") {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			continue
		}

		files = append(files, BackupFileInfo{
			Name:     name,
			Size:     info.Size(),
			ModTime:  info.ModTime().Format("2006-01-02 15:04:05"),
			FullPath: filepath.Join(dir, name),
		})
	}

	// 按修改时间倒序排列
	sort.Slice(files, func(i, j int) bool {
		return files[i].ModTime > files[j].ModTime
	})

	return files, nil
}

// GetCurrentFileInfo 获取当前日志文件信息
func (lw *LogFileWriter) GetCurrentFileInfo() (*BackupFileInfo, error) {
	info, err := os.Stat(lw.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return &BackupFileInfo{
				Name:     filepath.Base(lw.filePath),
				Size:     0,
				ModTime:  "",
				FullPath: lw.filePath,
			}, nil
		}
		return nil, err
	}

	return &BackupFileInfo{
		Name:     info.Name(),
		Size:     info.Size(),
		ModTime:  info.ModTime().Format("2006-01-02 15:04:05"),
		FullPath: lw.filePath,
	}, nil
}

// ClearLogFile 清空当前日志文件
func (lw *LogFileWriter) ClearLogFile() error {
	lw.mu.Lock()
	defer lw.mu.Unlock()

	// 关闭当前文件
	if lw.writer != nil {
		lw.writer.Flush()
	}
	if lw.file != nil {
		lw.file.Close()
		lw.file = nil
		lw.writer = nil
	}

	// 清空文件
	if err := os.Truncate(lw.filePath, 0); err != nil {
		if !os.IsNotExist(err) {
			return err
		}
	}

	// 重新打开
	return lw.openFile()
}

// UpdateConfig 更新配置（运行时动态修改）
func (lw *LogFileWriter) UpdateConfig(filePath, format, customTpl string, maxSize int64, maxBackups int, maxDays int, compress bool) {
	lw.mu.Lock()
	defer lw.mu.Unlock()

	pathChanged := lw.filePath != filePath

	lw.format = format
	lw.customTpl = customTpl
	lw.maxSize = maxSize
	lw.maxBackups = maxBackups
	lw.maxDays = maxDays
	lw.compress = compress
	lw.resolveTemplate()

	// 如果路径改变，重新打开文件
	if pathChanged {
		if lw.writer != nil {
			lw.writer.Flush()
		}
		if lw.file != nil {
			lw.file.Close()
			lw.file = nil
			lw.writer = nil
		}
		lw.filePath = filePath
		if err := lw.openFile(); err != nil {
			zlog.Error("更新配置后打开新日志文件失败: " + err.Error())
		}
	}
}
