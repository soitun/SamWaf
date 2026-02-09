package api

import (
	"SamWaf/global"
	"SamWaf/model/common/response"
	"SamWaf/wafnotify/logfilewriter"
	"github.com/gin-gonic/gin"
	"strconv"
)

type WafLogFileWriteApi struct {
}

// GetPreviewApi 获取日志文件预览（最新N行）
func (w *WafLogFileWriteApi) GetPreviewApi(c *gin.Context) {
	linesStr := c.DefaultQuery("lines", "100")
	lines, err := strconv.Atoi(linesStr)
	if err != nil || lines <= 0 {
		lines = 100
	}
	if lines > 500 {
		lines = 500
	}

	if global.GNOTIFY_LOG_FILE_WRITER == nil {
		response.FailWithMessage("日志文件写入服务未初始化", c)
		return
	}

	// 获取底层 notifier
	writer := getLogFileWriter()
	if writer == nil {
		response.FailWithMessage("日志文件写入服务未初始化", c)
		return
	}

	preview, err := writer.GetLogPreview(lines)
	if err != nil {
		response.FailWithMessage("获取日志预览失败: "+err.Error(), c)
		return
	}

	response.OkWithDetailed(preview, "获取成功", c)
}

// GetCurrentFileInfoApi 获取当前日志文件信息
func (w *WafLogFileWriteApi) GetCurrentFileInfoApi(c *gin.Context) {
	writer := getLogFileWriter()
	if writer == nil {
		response.FailWithMessage("日志文件写入服务未初始化", c)
		return
	}

	fileInfo, err := writer.GetCurrentFileInfo()
	if err != nil {
		response.FailWithMessage("获取文件信息失败: "+err.Error(), c)
		return
	}

	response.OkWithDetailed(fileInfo, "获取成功", c)
}

// GetBackupFilesApi 获取备份文件列表
func (w *WafLogFileWriteApi) GetBackupFilesApi(c *gin.Context) {
	writer := getLogFileWriter()
	if writer == nil {
		response.FailWithMessage("日志文件写入服务未初始化", c)
		return
	}

	files, err := writer.GetBackupFiles()
	if err != nil {
		response.FailWithMessage("获取备份文件列表失败: "+err.Error(), c)
		return
	}

	response.OkWithDetailed(files, "获取成功", c)
}

// ClearLogFileApi 清空当前日志文件
func (w *WafLogFileWriteApi) ClearLogFileApi(c *gin.Context) {
	writer := getLogFileWriter()
	if writer == nil {
		response.FailWithMessage("日志文件写入服务未初始化", c)
		return
	}

	err := writer.ClearLogFile()
	if err != nil {
		response.FailWithMessage("清空日志文件失败: "+err.Error(), c)
		return
	}

	response.OkWithMessage("清空成功", c)
}

// GetTemplateVariablesApi 获取可用的模板变量列表
func (w *WafLogFileWriteApi) GetTemplateVariablesApi(c *gin.Context) {
	variables := logfilewriter.GetTemplateVariables()
	response.OkWithDetailed(variables, "获取成功", c)
}

// getLogFileWriter 获取底层的 LogFileWriter
func getLogFileWriter() *logfilewriter.LogFileWriter {
	if global.GNOTIFY_LOG_FILE_WRITER == nil {
		return nil
	}
	notifier := global.GNOTIFY_LOG_FILE_WRITER.GetNotifier()
	if notifier == nil {
		return nil
	}
	if writer, ok := notifier.(*logfilewriter.LogFileWriter); ok {
		return writer
	}
	return nil
}
