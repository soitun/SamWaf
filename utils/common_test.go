package utils

import (
	"fmt"
	"log"
	"net"
	"strings"
	"testing"

	"github.com/oschwald/geoip2-golang"
)

func TestIPv6(t *testing.T) {
	db, err := geoip2.Open("../data/ipv6/GeoLite2-Country.mmdb")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	// If you are using strings that may be invalid, check that ip is not nil
	ip := net.ParseIP("2409:8087:3c02:21:0:1:0:100a")
	record, err := db.City(ip)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("国家: %v\n", record.Country.Names["zh-CN"])
	fmt.Sprintf("%v", record)
	fmt.Printf("Portuguese (BR) city name: %v\n", record.City.Names["pt-BR"])
	if len(record.Subdivisions) > 0 {
		fmt.Printf("English subdivision name: %v\n", record.Subdivisions[0].Names["en"])
	}
	fmt.Printf("Russian country name: %v\n", record.Country.Names["ru"])
	fmt.Printf("ISO country code: %v\n", record.Country.IsoCode)
	fmt.Printf("Time zone: %v\n", record.Location.TimeZone)
	fmt.Printf("Coordinates: %v, %v\n", record.Location.Latitude, record.Location.Longitude)
}

// TestIsValidIPv4 测试IPv4地址验证功能
func TestIsValidIPv4(t *testing.T) {
	tests := []struct {
		name     string
		ip       string
		expected bool
	}{
		{"valid IPv4", "192.168.1.1", true},
		{"valid IPv4 localhost", "127.0.0.1", true},
		{"valid IPv4 zero", "0.0.0.0", true},
		{"valid IPv4 max", "255.255.255.255", true},
		{"invalid IPv4 - too many octets", "192.168.1.1.1", false},
		{"invalid IPv4 - out of range", "256.256.256.256", false},
		{"invalid IPv4 - not a number", "abc.def.ghi.jkl", false},
		{"IPv6 address", "2001:0db8:85a3:0000:0000:8a2e:0370:7334", false},
		{"IPv6 address short", "240e:471:1a2b:3c4d:5e6f:7a8b:9c0d:9761", false},
		{"empty string", "", false},
		{"with port", "192.168.1.1:8080", false}, // IP地址不应该包含端口
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsValidIPv4(tt.ip)
			if result != tt.expected {
				t.Errorf("IsValidIPv4(%q) = %v, expected %v", tt.ip, result, tt.expected)
			}
		})
	}
}

// TestIsValidIPv6 测试IPv6地址验证功能
func TestIsValidIPv6(t *testing.T) {
	tests := []struct {
		name     string
		ip       string
		expected bool
	}{
		// 用户提供的IPv6地址
		{"user example 1", "240e:471:1a2b:3c4d:5e6f:7a8b:9c0d:9761", true},
		{"user example 2", "240e:471:9f8e:7d6c:5b4a:3210:abcd:9761", true},

		// 标准IPv6格式
		{"standard IPv6", "2001:0db8:85a3:0000:0000:8a2e:0370:7334", true},
		{"compressed IPv6", "2001:db8:85a3::8a2e:370:7334", true},
		{"loopback IPv6", "::1", true},
		{"unspecified IPv6", "::", true},

		// 无效的情况
		{"IPv4 address", "192.168.1.1", false},
		{"IPv4-mapped IPv6", "::ffff:192.168.1.1", false}, // IPv4映射地址会被To4()识别
		{"invalid IPv6 - too many groups", "2001:0db8:85a3:0000:0000:8a2e:0370:7334:1234", false},
		{"invalid IPv6 - not hex", "gggg:0db8:85a3:0000:0000:8a2e:0370:7334", false},
		{"empty string", "", false},
		{"with multiple IPs", "240e:471:1a2b:3c4d:5e6f:7a8b:9c0d:9761,240e:471:9f8e:7d6c:5b4a:3210:abcd:9761", false},
		{"with spaces", " 240e:471:1a2b:3c4d:5e6f:7a8b:9c0d:9761 ", false}, // 带空格应该失败
		{"with port", "[2001:db8::1]:8080", false},                         // 不应该包含端口
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsValidIPv6(tt.ip)
			if result != tt.expected {
				t.Errorf("IsValidIPv6(%q) = %v, expected %v", tt.ip, result, tt.expected)
			}
		})
	}
}

// TestIPParsing 测试IP地址解析，模拟getClientIP中的场景
func TestIPParsing(t *testing.T) {
	tests := []struct {
		name         string
		headerValue  string
		expectedIP   string
		shouldBeIPv4 bool
		shouldBeIPv6 bool
	}{
		{
			name:         "single IPv4",
			headerValue:  "192.168.1.1",
			expectedIP:   "192.168.1.1",
			shouldBeIPv4: true,
			shouldBeIPv6: false,
		},
		{
			name:         "multiple IPv4 with comma",
			headerValue:  "192.168.1.1, 10.0.0.1",
			expectedIP:   "192.168.1.1",
			shouldBeIPv4: true,
			shouldBeIPv6: false,
		},
		{
			name:         "single IPv6",
			headerValue:  "240e:471:1a2b:3c4d:5e6f:7a8b:9c0d:9761",
			expectedIP:   "240e:471:1a2b:3c4d:5e6f:7a8b:9c0d:9761",
			shouldBeIPv4: false,
			shouldBeIPv6: true,
		},
		{
			name:         "multiple IPv6 with comma (user scenario)",
			headerValue:  "240e:471:1a2b:3c4d:5e6f:7a8b:9c0d:9761,240e:471:9f8e:7d6c:5b4a:3210:abcd:9761",
			expectedIP:   "240e:471:1a2b:3c4d:5e6f:7a8b:9c0d:9761",
			shouldBeIPv4: false,
			shouldBeIPv6: true,
		},
		{
			name:         "IPv6 with spaces",
			headerValue:  " 240e:471:1a2b:3c4d:5e6f:7a8b:9c0d:9761 ",
			expectedIP:   "240e:471:1a2b:3c4d:5e6f:7a8b:9c0d:9761",
			shouldBeIPv4: false,
			shouldBeIPv6: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 模拟getClientIP中的逻辑
			// 分割逗号并取第一个IP
			ipParts := strings.Split(tt.headerValue, ",")
			trimmedIP := strings.TrimSpace(ipParts[0])

			isIPv4 := IsValidIPv4(trimmedIP)
			isIPv6 := IsValidIPv6(trimmedIP)

			if isIPv4 != tt.shouldBeIPv4 {
				t.Errorf("IsValidIPv4(%q) = %v, expected %v", trimmedIP, isIPv4, tt.shouldBeIPv4)
			}
			if isIPv6 != tt.shouldBeIPv6 {
				t.Errorf("IsValidIPv6(%q) = %v, expected %v", trimmedIP, isIPv6, tt.shouldBeIPv6)
			}
			if trimmedIP != tt.expectedIP {
				t.Errorf("Extracted IP = %q, expected %q", trimmedIP, tt.expectedIP)
			}
		})
	}
}
