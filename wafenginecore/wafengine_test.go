package wafenginecore

import (
	"SamWaf/common/zlog"
	"SamWaf/global"
	"bytes"
	"compress/flate"
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestReplaceBodyContent(t *testing.T) {
	// 测试用例表格
	tests := []struct {
		name        string
		inputBody   string
		oldString   string
		newString   string
		wantBody    string
		wantError   bool
		contentType string
	}{
		{
			name:      "basic replacement",
			inputBody: "Hello world",
			oldString: "world",
			newString: "golang",
			wantBody:  "Hello golang",
		},
		{
			name:      "multiple occurrences",
			inputBody: "banana",
			oldString: "na",
			newString: "no",
			wantBody:  "banono",
		},
		{
			name:      "empty body",
			inputBody: "",
			oldString: "test",
			newString: "demo",
			wantBody:  "",
		},
		{
			name:      "chinese characters",
			inputBody: "你好世界",
			oldString: "世界",
			newString: "Golang",
			wantBody:  "你好Golang",
		},
		{
			name:      "binary data",
			inputBody: string([]byte{0x48, 0x65, 0x00, 0x6c, 0x6c, 0x6f}), // 包含null字节
			oldString: string([]byte{0x00}),
			newString: " ",
			wantBody:  "He llo",
		},
		{
			name:      "case sensitive",
			inputBody: "Go is Cool, go is fun",
			oldString: "go",
			newString: "GO",
			wantBody:  "Go is Cool, GO is fun",
		},
		{
			name:        "json content",
			contentType: "application/json",
			inputBody:   `{"message":"hello"}`,
			oldString:   "hello",
			newString:   "hola",
			wantBody:    `{"message":"hola"}`,
		},
	}

	waf := &WafEngine{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 创建测试请求
			req := httptest.NewRequest("POST", "/", bytes.NewBufferString(tt.inputBody))
			if tt.contentType != "" {
				req.Header.Set("Content-Type", tt.contentType)
			}

			// 执行替换操作
			err := waf.ReplaceBodyContent(req, []string{tt.oldString}, tt.newString)
			if (err != nil) != tt.wantError {
				t.Fatalf("ReplaceBodyContent() error = %v, wantError %v", err, tt.wantError)
			}

			// 读取修改后的Body
			gotBody, err := io.ReadAll(req.Body)
			if err != nil {
				t.Fatal("Failed to read modified body:", err)
			}

			// 验证结果
			if string(gotBody) != tt.wantBody {
				t.Errorf("Body mismatch\nWant: %q\nGot:  %q", tt.wantBody, string(gotBody))
			}

			// 验证Content-Type是否保留
			if ct := req.Header.Get("Content-Type"); ct != tt.contentType {
				t.Errorf("Content-Type changed unexpectedly\nWas: %q\nNow: %q", tt.contentType, ct)
			}
		})
	}
}
func TestReplaceURLContent(t *testing.T) {
	t.Parallel()

	//初始化日志
	zlog.InitZLog(global.GWAF_LOG_DEBUG_ENABLE, "json")
	if v := recover(); v != nil {
		zlog.Error("error")
	}
	// 模拟原始请求
	req := httptest.NewRequest("GET", "http://origin.com/test%20/scan?q=hello%2520world&n=1%2B1", nil)
	waf := &WafEngine{}

	// 执行替换：将"test "替换为"demo"
	err := waf.ReplaceURLContent(req, []string{"test "}, "demo")
	if err != nil {
		t.Fatal(err)
	}

	// 验证路径编码
	if req.URL.Path != "/demo/scan" {
		t.Errorf("RawPath mismatch: %s", req.URL.RawPath)
	}

	// 模拟代理服务器接收的请求
	proxyReq := &http.Request{
		Method: req.Method,
		URL:    req.URL,
		Header: req.Header,
	}

	// 验证代理看到的URL
	if proxyReq.URL.String() != req.URL.String() {
		t.Errorf("Proxy URL mismatch: %s", proxyReq.URL.String())
	}
}

func TestGetOrgContent(t *testing.T) {
	t.Parallel()

	//初始化日志
	zlog.InitZLog(global.GWAF_LOG_DEBUG_ENABLE, "json")
	// 初始化WAF引擎
	waf := &WafEngine{}

	// 测试用例
	testCases := []struct {
		name            string
		contentType     string
		contentEncoding string
		content         string
		expectedContent string // 新增：期望解码后的内容
		expectedErr     bool
	}{
		{
			name:            "UTF-8 无压缩",
			contentType:     "text/html; charset=utf-8",
			contentEncoding: "",
			content:         "<html><body>这是UTF-8编码的内容</body></html>",
			expectedContent: "<html><body>这是UTF-8编码的内容</body></html>",
			expectedErr:     false,
		},
		{
			name:            "GBK 无压缩",
			contentType:     "text/html; charset=gbk",
			contentEncoding: "",
			// 这里使用GBK编码的字节序列，而不是UTF-8字符串
			content:         string([]byte{0x3c, 0x68, 0x74, 0x6d, 0x6c, 0x3e, 0x3c, 0x62, 0x6f, 0x64, 0x79, 0x3e, 0xd5, 0xe2, 0xca, 0xc7, 0x47, 0x42, 0x4b, 0xb1, 0xe0, 0xc2, 0xeb, 0xb5, 0xc4, 0xc4, 0xda, 0xc8, 0xdd, 0x3c, 0x2f, 0x62, 0x6f, 0x64, 0x79, 0x3e, 0x3c, 0x2f, 0x68, 0x74, 0x6d, 0x6c, 0x3e}),
			expectedContent: "<html><body>这是GBK编码的内容</body></html>",
			expectedErr:     false,
		},
		{
			name:            "UTF-8 GZIP压缩",
			contentType:     "text/html; charset=utf-8",
			contentEncoding: "gzip",
			content:         "<html><body>这是GZIP压缩的UTF-8内容</body></html>",
			expectedContent: "<html><body>这是GZIP压缩的UTF-8内容</body></html>",
			expectedErr:     false,
		},
		{
			name:            "UTF-8 Deflate压缩",
			contentType:     "text/html; charset=utf-8",
			contentEncoding: "deflate",
			content:         "<html><body>这是Deflate压缩的UTF-8内容</body></html>",
			expectedContent: "<html><body>这是Deflate压缩的UTF-8内容</body></html>",
			expectedErr:     false,
		},
		{
			name:            "无字符集指定",
			contentType:     "text/html",
			contentEncoding: "",
			content:         "<html><body>没有指定字符集的内容</body></html>",
			expectedContent: "<html><body>没有指定字符集的内容</body></html>",
			expectedErr:     true,
		},
		{
			// 对于不支持的字符集，我们跳过内容比较，只检查是否有错误
			name:            "不支持的字符集",
			contentType:     "text/html; charset=iso-8859-1",
			contentEncoding: "",
			content:         "<html><body>使用不常见字符集的内容</body></html>",
			expectedContent: "", // 不比较内容
			expectedErr:     false,
		},
		{
			name:            "JSON内容",
			contentType:     "application/json; charset=utf-8",
			contentEncoding: "",
			content:         `{"message": "这是JSON内容", "code": 200}`,
			expectedContent: `{"message": "这是JSON内容", "code": 200}`,
			expectedErr:     false,
		},
		{
			name:            "大量内容",
			contentType:     "text/html; charset=utf-8",
			contentEncoding: "",
			content:         strings.Repeat("这是一个很长的内容，用于测试大量数据的处理能力。", 10), // 减少重复次数以加快测试
			expectedContent: strings.Repeat("这是一个很长的内容，用于测试大量数据的处理能力。", 10),
			expectedErr:     false,
		},
	}

	for _, tc := range testCases {
		tc := tc // 防止闭包问题
		t.Run(tc.name, func(t *testing.T) {
			// 创建响应
			var bodyContent []byte
			var err error

			// 根据内容编码处理内容
			switch tc.contentEncoding {
			case "gzip":
				var buf bytes.Buffer
				gzipWriter := gzip.NewWriter(&buf)
				_, err = gzipWriter.Write([]byte(tc.content))
				if err != nil {
					t.Fatalf("创建gzip内容失败: %v", err)
				}
				gzipWriter.Close()
				bodyContent = buf.Bytes()
			case "deflate":
				var buf bytes.Buffer
				deflateWriter, _ := flate.NewWriter(&buf, flate.DefaultCompression)
				_, err = deflateWriter.Write([]byte(tc.content))
				if err != nil {
					t.Fatalf("创建deflate内容失败: %v", err)
				}
				deflateWriter.Close()
				bodyContent = buf.Bytes()
			default:
				bodyContent = []byte(tc.content)
			}

			// 创建HTTP响应
			resp := &http.Response{
				StatusCode: 200,
				Header:     make(http.Header),
				Body:       io.NopCloser(bytes.NewBuffer(bodyContent)),
			}
			resp.Header.Set("Content-Type", tc.contentType)
			if tc.contentEncoding != "" {
				resp.Header.Set("Content-Encoding", tc.contentEncoding)
			}

			// 调用测试函数
			result, _, err := waf.getOrgContent(resp, false, "")

			// 验证结果
			if tc.expectedErr {
				if err == nil {
					t.Errorf("期望错误但没有发生错误")
				}
			} else {
				if err != nil {
					t.Errorf("不期望错误但发生了错误: %v", err)
				} else if tc.expectedContent != "" { // 只有当期望内容不为空时才比较
					// 对于GBK测试用例，我们需要特殊处理
					if tc.name == "GBK 无压缩" {
						// 将结果转换为字符串并检查是否包含期望的文本
						resultStr := string(result)
						if !strings.Contains(resultStr, "这是GBK编码的内容") {
							t.Logf("原始内容: %s", tc.content)
							t.Logf("解码后内容: %s", resultStr)
							t.Errorf("GBK内容解码不正确")
						} else {
							t.Logf("测试通过: %s", tc.name)
						}
					} else {
						// 对于其他测试用例，直接比较内容
						if !strings.Contains(string(result), tc.expectedContent) {
							t.Logf("原始内容: %s", tc.content)
							t.Logf("解码后内容: %s", string(result))
							t.Errorf("内容解码不正确")
						} else {
							t.Logf("测试通过: %s", tc.name)
						}
					}
				} else {
					// 对于不比较内容的测试用例，只记录通过
					t.Logf("测试通过: %s (跳过内容比较)", tc.name)
				}
			}
		})
	}
}

// 测试Transfer-Encoding: chunked的情况
func TestGetOrgContentWithChunkedEncoding(t *testing.T) {
	t.Parallel()

	//初始化日志
	zlog.InitZLog(global.GWAF_LOG_DEBUG_ENABLE, "json")
	// 初始化WAF引擎
	waf := &WafEngine{}

	// 创建分块传输的内容
	chunkedContent := "10\r\n这是第一个数据块\r\n14\r\n这是第二个数据块内容\r\n0\r\n\r\n"

	// 创建HTTP响应
	resp := &http.Response{
		StatusCode:       200,
		Header:           make(http.Header),
		Body:             io.NopCloser(bytes.NewBufferString(chunkedContent)),
		TransferEncoding: []string{"chunked"},
	}
	resp.Header.Set("Content-Type", "text/html; charset=utf-8")

	// 调用测试函数
	result, _, err := waf.getOrgContent(resp, false, "")

	// 验证结果
	if err != nil {
		t.Errorf("处理chunked编码失败: %v", err)
	} else {
		t.Logf("Chunked编码处理结果: %s", string(result))
	}
}

// 测试响应体为空的情况
func TestGetOrgContentWithEmptyBody(t *testing.T) {
	t.Parallel()

	//初始化日志
	zlog.InitZLog(global.GWAF_LOG_DEBUG_ENABLE, "json")
	// 初始化WAF引擎
	waf := &WafEngine{}

	// 创建HTTP响应
	resp := &http.Response{
		StatusCode: 204, // No Content
		Header:     make(http.Header),
		Body:       io.NopCloser(bytes.NewBuffer(nil)),
	}
	resp.Header.Set("Content-Type", "text/html; charset=utf-8")

	// 调用测试函数
	result, _, err := waf.getOrgContent(resp, false, "")

	// 验证结果
	if err != nil {
		t.Errorf("处理空响应体失败: %v", err)
	} else {
		if len(result) != 0 {
			t.Errorf("空响应体应返回空内容，但返回了: %s", string(result))
		} else {
			t.Log("空响应体测试通过")
		}
	}
}

// 测试错误情况
func TestGetOrgContentWithErrors(t *testing.T) {
	t.Parallel()

	//初始化日志
	zlog.InitZLog(global.GWAF_LOG_DEBUG_ENABLE, "json")
	// 初始化WAF引擎
	waf := &WafEngine{}

	// 测试gzip解压失败
	invalidGzip := []byte{0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xff, 0x01, 0x00, 0x00, 0xff, 0xff, 0x00, 0x00, 0x00}
	resp := &http.Response{
		StatusCode: 200,
		Header:     make(http.Header),
		Body:       io.NopCloser(bytes.NewBuffer(invalidGzip)),
	}
	resp.Header.Set("Content-Type", "text/html; charset=utf-8")
	resp.Header.Set("Content-Encoding", "gzip")

	// 调用测试函数
	_, _, err := waf.getOrgContent(resp, false, "")

	// 验证结果
	if err == nil {
		t.Errorf("期望无效gzip内容产生错误，但没有错误")
	} else {
		t.Logf("无效gzip测试通过，错误: %v", err)
	}
}

func TestGetOrgContent_MetaAndDoctypeCharset(t *testing.T) {
	t.Parallel()
	zlog.InitZLog(global.GWAF_LOG_DEBUG_ENABLE, "json")
	waf := &WafEngine{}

	// 1. meta标签指定utf-8
	htmlMetaUtf8 := `<html><head><meta http-equiv="Content-Type" content="text/html; charset=utf-8"></head><body>utf8内容</body></html>`
	resp1 := &http.Response{
		StatusCode: 200,
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(htmlMetaUtf8)),
	}
	resp1.Header.Set("Content-Type", "text/html")
	content1, charset1, err1 := waf.getOrgContent(resp1, false, "")
	if err1 != nil || charset1 != "utf-8" || !strings.Contains(string(content1), "utf8内容") {
		t.Errorf("meta标签utf-8检测失败: err=%v, charset=%s, content=%s", err1, charset1, string(content1))
	}

	// 2. meta标签指定gbk
	gbkBody := []byte{0x3c, 0x68, 0x74, 0x6d, 0x6c, 0x3e, 0x3c, 0x68, 0x65, 0x61, 0x64, 0x3e, 0x3c, 0x6d, 0x65, 0x74, 0x61, 0x20, 0x68, 0x74, 0x74, 0x70, 0x2d, 0x65, 0x71, 0x75, 0x69, 0x76, 0x3d, 0x22, 0x43, 0x6f, 0x6e, 0x74, 0x65, 0x6e, 0x74, 0x2d, 0x54, 0x79, 0x70, 0x65, 0x22, 0x20, 0x63, 0x6f, 0x6e, 0x74, 0x65, 0x6e, 0x74, 0x3d, 0x22, 0x74, 0x65, 0x78, 0x74, 0x2f, 0x68, 0x74, 0x6d, 0x6c, 0x3b, 0x20, 0x63, 0x68, 0x61, 0x72, 0x73, 0x65, 0x74, 0x3d, 0x67, 0x62, 0x6b, 0x22, 0x3e, 0x3c, 0x2f, 0x68, 0x65, 0x61, 0x64, 0x3e, 0x3c, 0x62, 0x6f, 0x64, 0x79, 0x3e, 0xd5, 0xe2, 0xca, 0xc7, 0x47, 0x42, 0x4b, 0xb1, 0xe0, 0xc2, 0xeb, 0xb5, 0xc4, 0xc4, 0xda, 0xc8, 0xdd, 0x3c, 0x2f, 0x62, 0x6f, 0x64, 0x79, 0x3e, 0x3c, 0x2f, 0x68, 0x74, 0x6d, 0x6c, 0x3e}
	resp2 := &http.Response{
		StatusCode: 200,
		Header:     make(http.Header),
		Body:       io.NopCloser(bytes.NewReader(gbkBody)),
	}
	resp2.Header.Set("Content-Type", "text/html")
	content2, charset2, err2 := waf.getOrgContent(resp2, false, "")
	if err2 != nil || charset2 != "gbk" || !strings.Contains(string(content2), "这是GBK编码的内容") {
		t.Errorf("meta标签gbk检测失败: err=%v, charset=%s, content=%s", err2, charset2, string(content2))
	}

	// 3. DOCTYPE声明xhtml1-transitional.dtd
	htmlDoctype := `<!DOCTYPE html PUBLIC "-//W3C//DTD XHTML 1.0 Transitional//EN" "http://www.w3.org/TR/xhtml1/DTD/xhtml1-transitional.dtd"><html><head></head><body>doctype内容</body></html>`
	resp3 := &http.Response{
		StatusCode: 200,
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(htmlDoctype)),
	}
	resp3.Header.Set("Content-Type", "text/html")
	content3, charset3, err3 := waf.getOrgContent(resp3, false, "")
	if err3 != nil || charset3 != "utf-8" || !strings.Contains(string(content3), "doctype内容") {
		t.Errorf("DOCTYPE声明检测失败: err=%v, charset=%s, content=%s", err3, charset3, string(content3))
	}

	// 4. meta标签未知编码
	htmlMetaUnknown := `<html><head><meta charset="unknown-charset"></head><body>未知编码</body></html>`
	resp4 := &http.Response{
		StatusCode: 200,
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(htmlMetaUnknown)),
	}
	resp4.Header.Set("Content-Type", "text/html")
	_, _, err4 := waf.getOrgContent(resp4, false, "")
	if err4 == nil || !strings.Contains(err4.Error(), "编码检测不确定") {
		t.Errorf("未知meta编码检测失败，期望返回错误，实际: %v", err4)
	}
}

// TestShouldAutoJumpHTTPS 测试HTTPS自动跳转判断逻辑
func TestShouldAutoJumpHTTPS(t *testing.T) {
	// 初始化全局变量（如果有的话）
	global.GSSL_HTTP_CHANGLE_PATH = "/.well-known/acme-challenge/"

	tests := []struct {
		name           string
		requestHost    string
		configHost     string
		requestURL     string
		autoJumpHTTPS  int
		ssl            int
		expectedJump   bool
		expectedDomain string
		description    string
	}{
		// ========== 基本条件测试 ==========
		{
			name:           "未开启自动跳转",
			requestHost:    "example.com:80",
			configHost:     "example.com",
			requestURL:     "/test",
			autoJumpHTTPS:  0, // 未开启
			ssl:            1,
			expectedJump:   false,
			expectedDomain: "",
			description:    "autoJumpHTTPS=0，不应该跳转",
		},
		{
			name:           "未启用SSL",
			requestHost:    "example.com:80",
			configHost:     "example.com",
			requestURL:     "/test",
			autoJumpHTTPS:  1,
			ssl:            0, // 未启用SSL
			expectedJump:   false,
			expectedDomain: "",
			description:    "ssl=0，不应该跳转",
		},
		{
			name:           "SSL证书验证路径不跳转",
			requestHost:    "example.com:80",
			configHost:     "example.com",
			requestURL:     "/.well-known/acme-challenge/xxxxx",
			autoJumpHTTPS:  1,
			ssl:            1,
			expectedJump:   false,
			expectedDomain: "",
			description:    "SSL证书验证路径应排除",
		},

		// ========== 端口测试 ==========
		{
			name:           "标准80端口应跳转",
			requestHost:    "example.com:80",
			configHost:     "example.com",
			requestURL:     "/test",
			autoJumpHTTPS:  1,
			ssl:            1,
			expectedJump:   true,
			expectedDomain: "example.com",
			description:    "标准80端口应触发跳转",
		},
		{
			name:           "非标准8080端口应跳转",
			requestHost:    "example.com:8080",
			configHost:     "example.com",
			requestURL:     "/test",
			autoJumpHTTPS:  1,
			ssl:            1,
			expectedJump:   true,
			expectedDomain: "example.com",
			description:    "8080端口应触发跳转",
		},
		{
			name:           "非标准8888端口应跳转",
			requestHost:    "example.com:8888",
			configHost:     "example.com",
			requestURL:     "/test",
			autoJumpHTTPS:  1,
			ssl:            1,
			expectedJump:   true,
			expectedDomain: "example.com",
			description:    "8888端口应触发跳转",
		},
		{
			name:           "443端口不应跳转",
			requestHost:    "example.com:443",
			configHost:     "example.com",
			requestURL:     "/test",
			autoJumpHTTPS:  1,
			ssl:            1,
			expectedJump:   false,
			expectedDomain: "",
			description:    "443端口已是HTTPS，不应跳转",
		},
		{
			name:           "无端口不应跳转",
			requestHost:    "example.com",
			configHost:     "example.com",
			requestURL:     "/test",
			autoJumpHTTPS:  1,
			ssl:            1,
			expectedJump:   false,
			expectedDomain: "",
			description:    "无端口号不应跳转",
		},

		// ========== 精确匹配测试 ==========
		{
			name:           "精确匹配-域名完全一致",
			requestHost:    "example.com:80",
			configHost:     "example.com",
			requestURL:     "/test",
			autoJumpHTTPS:  1,
			ssl:            1,
			expectedJump:   true,
			expectedDomain: "example.com",
			description:    "请求域名与配置域名完全一致应跳转",
		},
		{
			name:           "精确匹配-域名不一致",
			requestHost:    "test.com:80",
			configHost:     "example.com",
			requestURL:     "/test",
			autoJumpHTTPS:  1,
			ssl:            1,
			expectedJump:   false,
			expectedDomain: "",
			description:    "请求域名与配置域名不一致不应跳转",
		},
		{
			name:           "带端口的精确匹配",
			requestHost:    "example.com:8080",
			configHost:     "example.com:8080",
			requestURL:     "/test",
			autoJumpHTTPS:  1,
			ssl:            1,
			expectedJump:   true,
			expectedDomain: "example.com",
			description:    "配置host带端口且完全匹配应跳转",
		},

		// ========== 二级泛域名测试 ==========
		{
			name:           "二级泛域名-正常匹配",
			requestHost:    "aaa.samwaf.com:80",
			configHost:     "*.samwaf.com",
			requestURL:     "/test",
			autoJumpHTTPS:  1,
			ssl:            1,
			expectedJump:   true,
			expectedDomain: "aaa.samwaf.com",
			description:    "二级域名应匹配泛域名配置",
		},
		{
			name:           "二级泛域名-非标准端口",
			requestHost:    "bbb.samwaf.com:8080",
			configHost:     "*.samwaf.com",
			requestURL:     "/test",
			autoJumpHTTPS:  1,
			ssl:            1,
			expectedJump:   true,
			expectedDomain: "bbb.samwaf.com",
			description:    "二级域名+非标准端口应匹配泛域名配置",
		},
		{
			name:           "二级泛域名-根域名不应匹配",
			requestHost:    "samwaf.com:80",
			configHost:     "*.samwaf.com",
			requestURL:     "/test",
			autoJumpHTTPS:  1,
			ssl:            1,
			expectedJump:   false,
			expectedDomain: "",
			description:    "根域名不应匹配*.samwaf.com",
		},
		{
			name:           "二级泛域名-部分匹配不应成功",
			requestHost:    "aaasamwaf.com:80",
			configHost:     "*.samwaf.com",
			requestURL:     "/test",
			autoJumpHTTPS:  1,
			ssl:            1,
			expectedJump:   false,
			expectedDomain: "",
			description:    "aaasamwaf.com不应匹配*.samwaf.com",
		},

		// ========== 三级泛域名测试 ==========
		{
			name:           "三级泛域名-正常匹配",
			requestHost:    "bbb.aaa.samwaf.com:80",
			configHost:     "*.samwaf.com",
			requestURL:     "/test",
			autoJumpHTTPS:  1,
			ssl:            1,
			expectedJump:   true,
			expectedDomain: "bbb.aaa.samwaf.com",
			description:    "三级域名应匹配泛域名配置（通过MaskSubdomain）",
		},
		{
			name:           "三级泛域名-非标准端口",
			requestHost:    "ccc.bbb.samwaf.com:8080",
			configHost:     "*.samwaf.com",
			requestURL:     "/test",
			autoJumpHTTPS:  1,
			ssl:            1,
			expectedJump:   true,
			expectedDomain: "ccc.bbb.samwaf.com",
			description:    "三级域名+非标准端口应匹配泛域名配置",
		},

		// ========== 四级泛域名测试 ==========
		{
			name:           "四级泛域名-正常匹配",
			requestHost:    "ddd.ccc.bbb.samwaf.com:80",
			configHost:     "*.samwaf.com",
			requestURL:     "/test",
			autoJumpHTTPS:  1,
			ssl:            1,
			expectedJump:   true,
			expectedDomain: "ddd.ccc.bbb.samwaf.com",
			description:    "四级域名应匹配泛域名配置",
		},
		{
			name:           "四级泛域名-非标准端口8888",
			requestHost:    "eee.ddd.ccc.samwaf.com:8888",
			configHost:     "*.samwaf.com",
			requestURL:     "/test",
			autoJumpHTTPS:  1,
			ssl:            1,
			expectedJump:   true,
			expectedDomain: "eee.ddd.ccc.samwaf.com",
			description:    "四级域名+8888端口应匹配泛域名配置",
		},

		// ========== 边界情况测试 ==========
		{
			name:           "空路径",
			requestHost:    "example.com:80",
			configHost:     "example.com",
			requestURL:     "",
			autoJumpHTTPS:  1,
			ssl:            1,
			expectedJump:   true,
			expectedDomain: "example.com",
			description:    "空路径应正常跳转",
		},
		{
			name:           "根路径",
			requestHost:    "example.com:80",
			configHost:     "example.com",
			requestURL:     "/",
			autoJumpHTTPS:  1,
			ssl:            1,
			expectedJump:   true,
			expectedDomain: "example.com",
			description:    "根路径应正常跳转",
		},
		{
			name:           "带查询参数的路径",
			requestHost:    "example.com:80",
			configHost:     "example.com",
			requestURL:     "/test?param=value",
			autoJumpHTTPS:  1,
			ssl:            1,
			expectedJump:   true,
			expectedDomain: "example.com",
			description:    "带查询参数应正常跳转",
		},

		// ========== 不同域名后缀测试 ==========
		{
			name:           "不同域名后缀-com",
			requestHost:    "test.example.com:80",
			configHost:     "*.example.com",
			requestURL:     "/test",
			autoJumpHTTPS:  1,
			ssl:            1,
			expectedJump:   true,
			expectedDomain: "test.example.com",
			description:    ".com域名应正常匹配",
		},
		{
			name:           "不同域名后缀-cn",
			requestHost:    "test.example.cn:80",
			configHost:     "*.example.cn",
			requestURL:     "/test",
			autoJumpHTTPS:  1,
			ssl:            1,
			expectedJump:   true,
			expectedDomain: "test.example.cn",
			description:    ".cn域名应正常匹配",
		},
		{
			name:           "不同域名后缀-io",
			requestHost:    "api.service.io:8080",
			configHost:     "*.service.io",
			requestURL:     "/api/v1",
			autoJumpHTTPS:  1,
			ssl:            1,
			expectedJump:   true,
			expectedDomain: "api.service.io",
			description:    ".io域名+非标准端口应正常匹配",
		},

		// ========== 特殊字符测试 ==========
		{
			name:           "域名含中划线",
			requestHost:    "test-api.example.com:80",
			configHost:     "*.example.com",
			requestURL:     "/test",
			autoJumpHTTPS:  1,
			ssl:            1,
			expectedJump:   true,
			expectedDomain: "test-api.example.com",
			description:    "域名包含中划线应正常匹配",
		},
		{
			name:           "域名含数字",
			requestHost:    "api123.example.com:80",
			configHost:     "*.example.com",
			requestURL:     "/test",
			autoJumpHTTPS:  1,
			ssl:            1,
			expectedJump:   true,
			expectedDomain: "api123.example.com",
			description:    "域名包含数字应正常匹配",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotJump, gotDomain := shouldAutoJumpHTTPS(
				tt.requestHost,
				tt.configHost,
				tt.requestURL,
				tt.autoJumpHTTPS,
				tt.ssl,
			)

			if gotJump != tt.expectedJump {
				t.Errorf("%s\n请求Host: %s, 配置Host: %s\n期望跳转: %v, 实际跳转: %v",
					tt.description, tt.requestHost, tt.configHost, tt.expectedJump, gotJump)
			}

			if gotJump && gotDomain != tt.expectedDomain {
				t.Errorf("%s\n请求Host: %s, 配置Host: %s\n期望域名: %s, 实际域名: %s",
					tt.description, tt.requestHost, tt.configHost, tt.expectedDomain, gotDomain)
			}

			// 成功的测试用例也输出日志
			if gotJump == tt.expectedJump {
				t.Logf("✅ %s - 通过", tt.name)
			}
		})
	}
}

// TestShouldAutoJumpHTTPS_Comprehensive 综合场景测试
func TestShouldAutoJumpHTTPS_Comprehensive(t *testing.T) {
	global.GSSL_HTTP_CHANGLE_PATH = "/.well-known/acme-challenge/"

	// 场景1: 用户反馈的实际问题场景
	t.Run("用户反馈场景1-非标准80端口", func(t *testing.T) {
		// 场景：用户使用8080端口，之前无法跳转
		jump, domain := shouldAutoJumpHTTPS("example.com:8080", "example.com", "/test", 1, 1)
		if !jump || domain != "example.com" {
			t.Errorf("非标准80端口应该能跳转，实际: jump=%v, domain=%s", jump, domain)
		} else {
			t.Logf("✅ 非标准80端口测试通过")
		}
	})

	t.Run("用户反馈场景2-泛域名匹配", func(t *testing.T) {
		// 场景：host变量: aaa.samwaf.com:80, hostTarget.Host.Host变量: *.samwaf.com
		jump, domain := shouldAutoJumpHTTPS("aaa.samwaf.com:80", "*.samwaf.com", "/test", 1, 1)
		if !jump || domain != "aaa.samwaf.com" {
			t.Errorf("泛域名应该能匹配，实际: jump=%v, domain=%s", jump, domain)
		} else {
			t.Logf("✅ 泛域名匹配测试通过")
		}
	})

	t.Run("用户反馈场景3-三级泛域名", func(t *testing.T) {
		// 场景：bbb.aaa.samwaf.com:80 匹配 *.samwaf.com
		jump, domain := shouldAutoJumpHTTPS("bbb.aaa.samwaf.com:80", "*.samwaf.com", "/test", 1, 1)
		if !jump || domain != "bbb.aaa.samwaf.com" {
			t.Errorf("三级泛域名应该能匹配，实际: jump=%v, domain=%s", jump, domain)
		} else {
			t.Logf("✅ 三级泛域名匹配测试通过")
		}
	})

	// 场景2: SSL证书自动续期路径
	t.Run("SSL证书续期路径不应跳转", func(t *testing.T) {
		paths := []string{
			"/.well-known/acme-challenge/test123",
			"/.well-known/acme-challenge/",
		}
		for _, path := range paths {
			jump, _ := shouldAutoJumpHTTPS("example.com:80", "example.com", path, 1, 1)
			if jump {
				t.Errorf("SSL证书路径 %s 不应跳转", path)
			} else {
				t.Logf("✅ SSL证书路径 %s 正确排除", path)
			}
		}
	})

	// 场景3: 多个非标准端口
	t.Run("各种非标准HTTP端口", func(t *testing.T) {
		ports := []string{"80", "8080", "8888", "9090", "3000", "5000"}
		for _, port := range ports {
			requestHost := "example.com:" + port
			jump, domain := shouldAutoJumpHTTPS(requestHost, "example.com", "/test", 1, 1)
			if !jump || domain != "example.com" {
				t.Errorf("端口 %s 应该能跳转，实际: jump=%v, domain=%s", port, jump, domain)
			} else {
				t.Logf("✅ 端口 %s 测试通过", port)
			}
		}
	})

	// 场景4: 多级域名递归测试
	t.Run("多级域名递归匹配", func(t *testing.T) {
		testCases := []struct {
			requestHost string
			configHost  string
		}{
			{"a.samwaf.com:80", "*.samwaf.com"},
			{"b.a.samwaf.com:80", "*.samwaf.com"},
			{"c.b.a.samwaf.com:80", "*.samwaf.com"},
			{"d.c.b.a.samwaf.com:80", "*.samwaf.com"},
			{"e.d.c.b.a.samwaf.com:80", "*.samwaf.com"},
		}

		for _, tc := range testCases {
			jump, domain := shouldAutoJumpHTTPS(tc.requestHost, tc.configHost, "/test", 1, 1)
			expectedDomain := strings.Split(tc.requestHost, ":")[0]
			if !jump || domain != expectedDomain {
				t.Errorf("多级域名 %s 匹配 %s 失败，实际: jump=%v, domain=%s",
					tc.requestHost, tc.configHost, jump, domain)
			} else {
				t.Logf("✅ 多级域名 %s 匹配成功", tc.requestHost)
			}
		}
	})
}
