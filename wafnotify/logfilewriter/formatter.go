package logfilewriter

import (
	"SamWaf/innerbean"
	"strconv"
	"strings"
)

// 预定义格式模板
const (
	// NginxCombinedFormat nginx combined 格式
	NginxCombinedFormat = `${src_ip} - - [${create_time}] "${method} ${url} HTTP/1.1" ${status_code} ${content_length} "${referer}" "${user_agent}"`

	// ApacheCombinedFormat apache combined 格式
	ApacheCombinedFormat = `${src_ip} - - [${create_time}] "${method} ${url} HTTP/1.1" ${status_code} ${content_length} "${referer}" "${user_agent}"`
)

// GetFormatTemplate 根据格式名称获取模板
func GetFormatTemplate(format string) string {
	switch strings.ToLower(format) {
	case "nginx":
		return NginxCombinedFormat
	case "apache":
		return ApacheCombinedFormat
	case "custom":
		return "" // 自定义格式由用户提供
	default:
		return NginxCombinedFormat
	}
}

// FormatLog 将 WebLog 格式化为指定模板的字符串
func FormatLog(log *innerbean.WebLog, template string) string {
	if template == "" {
		template = NginxCombinedFormat
	}

	result := template

	// 替换所有模板变量
	result = strings.ReplaceAll(result, "${src_ip}", log.SRC_IP)
	result = strings.ReplaceAll(result, "${src_port}", log.SRC_PORT)
	result = strings.ReplaceAll(result, "${host}", log.HOST)
	result = strings.ReplaceAll(result, "${url}", log.URL)
	result = strings.ReplaceAll(result, "${raw_query}", log.RawQuery)
	result = strings.ReplaceAll(result, "${method}", log.METHOD)
	result = strings.ReplaceAll(result, "${scheme}", log.Scheme)
	result = strings.ReplaceAll(result, "${referer}", log.REFERER)
	result = strings.ReplaceAll(result, "${user_agent}", log.USER_AGENT)
	result = strings.ReplaceAll(result, "${status_code}", strconv.Itoa(log.STATUS_CODE))
	result = strings.ReplaceAll(result, "${content_length}", strconv.FormatInt(log.CONTENT_LENGTH, 10))
	result = strings.ReplaceAll(result, "${create_time}", log.CREATE_TIME)
	result = strings.ReplaceAll(result, "${time_spent}", strconv.FormatInt(log.TimeSpent, 10))
	result = strings.ReplaceAll(result, "${country}", log.COUNTRY)
	result = strings.ReplaceAll(result, "${province}", log.PROVINCE)
	result = strings.ReplaceAll(result, "${city}", log.CITY)
	result = strings.ReplaceAll(result, "${action}", log.ACTION)
	result = strings.ReplaceAll(result, "${rule}", log.RULE)
	result = strings.ReplaceAll(result, "${risk_level}", strconv.Itoa(log.RISK_LEVEL))
	result = strings.ReplaceAll(result, "${cookies}", log.COOKIES)
	result = strings.ReplaceAll(result, "${req_uuid}", log.REQ_UUID)
	result = strings.ReplaceAll(result, "${is_bot}", strconv.Itoa(log.IsBot))
	result = strings.ReplaceAll(result, "${guest_identification}", log.GUEST_IDENTIFICATION)

	return result
}

// TemplateVariables 返回所有可用的模板变量信息（供前端展示）
type TemplateVariable struct {
	Name    string `json:"name"`    // 变量名
	Field   string `json:"field"`   // 对应字段
	Desc    string `json:"desc"`    // 说明
	Example string `json:"example"` // 示例值
}

// GetTemplateVariables 返回所有可用的模板变量
func GetTemplateVariables() []TemplateVariable {
	return []TemplateVariable{
		{Name: "${src_ip}", Field: "SRC_IP", Desc: "客户端IP", Example: "192.168.1.100"},
		{Name: "${src_port}", Field: "SRC_PORT", Desc: "客户端端口", Example: "52341"},
		{Name: "${host}", Field: "HOST", Desc: "请求主机名", Example: "www.example.com"},
		{Name: "${url}", Field: "URL", Desc: "请求URL", Example: "/api/user?id=1"},
		{Name: "${raw_query}", Field: "RawQuery", Desc: "URL查询参数", Example: "id=1&name=test"},
		{Name: "${method}", Field: "METHOD", Desc: "请求方法", Example: "GET"},
		{Name: "${scheme}", Field: "Scheme", Desc: "协议", Example: "https"},
		{Name: "${referer}", Field: "REFERER", Desc: "Referer头", Example: "https://example.com/"},
		{Name: "${user_agent}", Field: "USER_AGENT", Desc: "User-Agent", Example: "Mozilla/5.0..."},
		{Name: "${status_code}", Field: "STATUS_CODE", Desc: "HTTP状态码", Example: "200"},
		{Name: "${content_length}", Field: "CONTENT_LENGTH", Desc: "响应大小", Example: "2048"},
		{Name: "${create_time}", Field: "CREATE_TIME", Desc: "请求时间", Example: "2026-02-09 10:00:00"},
		{Name: "${time_spent}", Field: "TimeSpent", Desc: "耗时(ms)", Example: "125"},
		{Name: "${country}", Field: "COUNTRY", Desc: "国家", Example: "中国"},
		{Name: "${province}", Field: "PROVINCE", Desc: "省份", Example: "广东省"},
		{Name: "${city}", Field: "CITY", Desc: "城市", Example: "深圳市"},
		{Name: "${action}", Field: "ACTION", Desc: "防御动作", Example: "放行"},
		{Name: "${rule}", Field: "RULE", Desc: "触发规则", Example: "SQL注入检测"},
		{Name: "${risk_level}", Field: "RISK_LEVEL", Desc: "风险等级", Example: "0"},
		{Name: "${cookies}", Field: "COOKIES", Desc: "Cookies", Example: "session=abc123"},
		{Name: "${req_uuid}", Field: "REQ_UUID", Desc: "请求ID", Example: "a1b2c3d4-..."},
		{Name: "${is_bot}", Field: "IsBot", Desc: "是否机器人", Example: "0"},
		{Name: "${guest_identification}", Field: "GUEST_IDENTIFICATION", Desc: "访客标识", Example: "Googlebot"},
	}
}
