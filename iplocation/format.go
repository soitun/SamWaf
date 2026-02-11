package iplocation

import "strings"

// ParseRegion 根据 DBFormat 将 "|" 分隔的 region 字符串解析为统一的 IPLocationResult
func ParseRegion(region string, format DBFormat) *IPLocationResult {
	if region == "" {
		return &IPLocationResult{}
	}

	fields := strings.Split(region, "|")

	// 辅助函数：安全获取字段，处理 "0" 和空值
	getField := func(index int) string {
		if index >= len(fields) {
			return ""
		}
		val := strings.TrimSpace(fields[index])
		if val == "0" || val == "" {
			return ""
		}
		return val
	}

	result := &IPLocationResult{}

	switch format {
	case FormatLegacy: // 国家|区域|省份|城市|ISP
		result.Country = getField(0)
		result.Region = getField(1)
		result.Province = getField(2)
		result.City = getField(3)
		result.ISP = getField(4)

	case FormatOpenSource: // 国家|省份|城市|网络运营商
		result.Country = getField(0)
		result.Province = getField(1)
		result.City = getField(2)
		result.ISP = getField(3)

	case FormatFull: // 大洲|国家|省份|城市|区县|网络运营商|其他
		result.Region = getField(0)
		result.Country = getField(1)
		result.Province = getField(2)
		result.City = getField(3)
		result.District = getField(4)
		result.ISP = getField(5)

	case FormatStandard: // 国家|省份|城市|区县|网络运营商|其他
		result.Country = getField(0)
		result.Province = getField(1)
		result.City = getField(2)
		result.District = getField(3)
		result.ISP = getField(4)

	case FormatCompact: // 国家|省份|城市|网络运营商|其他
		result.Country = getField(0)
		result.Province = getField(1)
		result.City = getField(2)
		result.ISP = getField(3)

	default:
		// 默认按 legacy 格式处理
		result.Country = getField(0)
		result.Region = getField(1)
		result.Province = getField(2)
		result.City = getField(3)
		result.ISP = getField(4)
	}

	// 处理特殊标记 "内网IP"
	if len(fields) > 0 {
		lastField := fields[len(fields)-1]
		if strings.Contains(lastField, "内网IP") {
			result.Country = "内网"
			result.Province = "内网"
			result.City = "内网"
			result.Region = ""
			result.ISP = ""
		}
	}

	return result
}
