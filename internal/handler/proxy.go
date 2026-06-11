package handler

import (
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"
)

// ProxyHandler 是一个 HTTP 代理处理器，用于在 iframe 中嵌入外部网页。
//
//	GET /api/proxy?url=https://example.com
//
// 为什么需要代理？
//
//	现代浏览器有同源策略（Same-Origin Policy）和 X-Frame-Options 限制，
//	很多网站不允许被其他域名的页面通过 iframe 嵌入。
//	通过服务端代理抓取页面内容，可以绕过这些限制。
//
// 代理做了什么：
//  1. 接收前端传过来的目标 URL
//  2. 服务端去请求这个 URL（服务端不受浏览器同源策略限制）
//  3. 对 HTML 页面注入 <base> 标签，让相对路径的资源能正确加载
//  4. 把结果返回给前端
//
// 注意这个接口不需要登录认证，因为它是为 iframe 嵌入设计的。
func ProxyHandler(c *gin.Context) {
	// Step 1: 获取目标 URL
	targetURL := c.Query("url")
	if targetURL == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "url parameter required"})
		return
	}

	// Step 2: 校验 URL 格式
	// url.ParseRequestURI 会检查 URL 是否合法（有 scheme、host 等）
	parsed, err := url.ParseRequestURI(targetURL)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid url"})
		return
	}

	// Step 3: 服务端发起 HTTP GET 请求
	// 注意：这里用的 http.Get 会跟随重定向（最多 10 次）
	resp, err := http.Get(targetURL)
	if err != nil {
		// HTTP 502 Bad Gateway：代理请求上游失败
		c.JSON(http.StatusBadGateway, gin.H{"error": "failed to fetch url"})
		return
	}
	defer resp.Body.Close() // 函数结束时关闭响应体

	// Step 4: 透传关键响应头
	contentType := resp.Header.Get("Content-Type")

	// 只透传 Content-Type 和 Content-Length，其他头部可能不安全
	for _, h := range []string{"Content-Type", "Content-Length"} {
		if v := resp.Header.Get(h); v != "" {
			c.Header(h, v)
		}
	}

	// Step 5: 读取完整的响应体
	body, _ := io.ReadAll(resp.Body)

	// Step 6: 对 HTML 页面注入 <base> 标签
	//
	// 为什么要注入 <base> 标签？
	//   HTML 页面中的图片、CSS、JS 等资源经常使用相对路径，如：
	//   <img src="/images/logo.png">
	//   如果通过代理返回，浏览器会按代理的域名来解析相对路径，导致 404。
	//   注入 <base href="原始URL"> 后，浏览器会按原始域解析相对路径。
	//
	// 例如：
	//   代理 URL:  http://localhost:8080/api/proxy?url=https://example.com/page
	//   页面中有:  <img src="/images/logo.png">
	//   注入后:    <base href="https://example.com">
	//   浏览器会向 https://example.com/images/logo.png 请求图片 ✓
	if strings.Contains(contentType, "text/html") {
		baseTag := "<base href=\"" + targetURL + "\">"
		// 把 <base> 标签插入到 <head> 标签内部的开头位置
		// strings.Replace 的最后一个参数 1 表示只替换第一个匹配
		body = []byte(strings.Replace(string(body), "<head>", "<head>"+baseTag, 1))
		// 清除 Content-Length（因为注入 base 标签后长度变了）
		c.Header("Content-Length", "")
	}

	// Step 7: 设置代理基础 URL，前端可能需要用到
	baseURL := parsed.Scheme + "://" + parsed.Host
	c.Header("X-Proxy-Base", baseURL)

	// Step 8: 返回代理结果
	// c.Data 直接返回原始字节数据 + 指定的 Content-Type 和状态码
	c.Data(resp.StatusCode, contentType, body)
}
