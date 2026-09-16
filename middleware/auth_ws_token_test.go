package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

// 管理端 WebSocket 的鉴权来源回归用例。
//
// 背景：WS 与下载链接的令牌都不在常规请求头里，中间件靠请求路径来决定去哪儿取。
// 这个判定曾经比的是 RequestURI（带查询串），于是 `/api/v1/ws?keyid=xxx` 一加参数就失配，
// 落到常规请求头分支取到空令牌 → 连接被判鉴权失败 → 页面收不到任何通知。
// 下面的用例把"带查询串"这件事钉死。

func ctxFor(t *testing.T, method, target string, headers map[string]string) *gin.Context {
	t.Helper()
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	req := httptest.NewRequest(method, target, nil)
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	c.Request = req
	return c
}

func TestExtractTokenStrWebSocket(t *testing.T) {
	cases := []struct {
		name   string
		target string
	}{
		{"无查询串", "/api/v1/ws"},
		{"带会话密钥标识", "/api/v1/ws?keyid=w72WNCyF63z26BnH5_nSzw"},
		{"多个查询参数", "/api/v1/ws?keyid=abc&foo=bar"},
		{"查询串为空", "/api/v1/ws?"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := ctxFor(t, http.MethodGet, tc.target, map[string]string{
				"Sec-WebSocket-Protocol": "the-token",
			})
			if got := extractTokenStr(c); got != "the-token" {
				t.Fatalf("WebSocket 令牌应取自 Sec-WebSocket-Protocol，实际取到 %q", got)
			}
		})
	}
}

// WebSocket 握手带不了自定义头，所以哪怕 X-Token 存在也不能用它来判定：
// 真实浏览器发起的 WS 请求里根本不会有这个头，测试里若误取会给出假阳性。
func TestExtractTokenStrWebSocketIgnoresXToken(t *testing.T) {
	c := ctxFor(t, http.MethodGet, "/api/v1/ws?keyid=abc", map[string]string{
		"Sec-WebSocket-Protocol": "ws-token",
		"X-Token":                "header-token",
	})
	if got := extractTokenStr(c); got != "ws-token" {
		t.Fatalf("期望 ws-token，实际 %q", got)
	}
}

func TestExtractTokenStrDownload(t *testing.T) {
	c := ctxFor(t, http.MethodGet,
		"/api/v1/waflog/attack/download?X-Token=dl-token&X-Request-Id=abc", nil)
	if got := extractTokenStr(c); got != "dl-token" {
		t.Fatalf("下载链接令牌应取自查询串，实际 %q", got)
	}
}

func TestExtractTokenStrNormalAndMobile(t *testing.T) {
	web := ctxFor(t, http.MethodPost, "/api/v1/hosts/list?page=1", map[string]string{
		"X-Token": "web-token",
	})
	if got := extractTokenStr(web); got != "web-token" {
		t.Fatalf("普通请求应取 X-Token，实际 %q", got)
	}

	mobile := ctxFor(t, http.MethodPost, "/api/v1/hosts/list", map[string]string{
		"X-Login-Type":   "mobile",
		"X-Mobile-Token": "mobile-token",
		"X-Token":        "web-token",
	})
	if got := extractTokenStr(mobile); got != "mobile-token" {
		t.Fatalf("移动端应取 X-Mobile-Token，实际 %q", got)
	}
}

// 设备指纹豁免同样按路径判定，不能被查询串带偏——
// 否则 WS 会去比对浏览器根本不会发的那几个头，必然误杀。
func TestFingerprintExemptWithQueryString(t *testing.T) {
	cases := []struct {
		name   string
		target string
		header map[string]string
		want   bool
	}{
		{"ws 带查询串", "/api/v1/ws?keyid=abc", nil, true},
		{"ws 无查询串", "/api/v1/ws", nil, true},
		{"下载带查询串", "/api/v1/waflog/attack/download?X-Token=t", nil, true},
		{"Upgrade 头", "/whatever", map[string]string{"Upgrade": "websocket"}, true},
		{"SSE", "/api/v1/gpt/chat", map[string]string{"Accept": "text/event-stream"}, true},
		{"普通接口", "/api/v1/hosts/list?page=1", nil, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := ctxFor(t, http.MethodGet, tc.target, tc.header)
			if got := isFingerprintExemptPath(c); got != tc.want {
				t.Fatalf("期望 %v，实际 %v", tc.want, got)
			}
		})
	}
}

// 未登录时前端从 localStorage 取不到令牌，浏览器会把 JS 的 null 原样字符串化成 "null"
// 放进子协议发出来。不归一的话它是个非空字符串，会一路走到缓存查询并被判成"令牌不存在"，
// 登录页停着不动也会每隔几秒刷一条无效令牌日志。
func TestExtractTokenStrTreatsNullLiteralAsEmpty(t *testing.T) {
	cases := []struct {
		name   string
		target string
		header string
		value  string
	}{
		{"WebSocket 子协议 null", "/api/v1/ws", "Sec-WebSocket-Protocol", "null"},
		{"WebSocket 子协议 undefined", "/api/v1/ws", "Sec-WebSocket-Protocol", "undefined"},
		{"WebSocket 子协议空白", "/api/v1/ws", "Sec-WebSocket-Protocol", "  "},
		{"常规请求头 null", "/api/v1/host/list", "X-Token", "null"},
		{"常规请求头 undefined", "/api/v1/host/list", "X-Token", "undefined"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := ctxFor(t, http.MethodGet, tc.target, map[string]string{tc.header: tc.value})
			if got := extractTokenStr(c); got != "" {
				t.Fatalf("%q 应被当作没有令牌，实际取到 %q", tc.value, got)
			}
		})
	}
}

// 归一化不能误伤正常令牌
func TestExtractTokenStrKeepsRealToken(t *testing.T) {
	c := ctxFor(t, http.MethodGet, "/api/v1/host/list", map[string]string{
		"X-Token": "0d42ce0c1f2a3b4c5d6e7f8091a2b3c4",
	})
	if got := extractTokenStr(c); got != "0d42ce0c1f2a3b4c5d6e7f8091a2b3c4" {
		t.Fatalf("正常令牌被改动了：%q", got)
	}
}
