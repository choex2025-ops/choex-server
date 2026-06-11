package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// CORS 返回一个跨域资源共享（CORS）中间件。
//
// CORS 是什么？
//
//	浏览器的同源策略（Same-Origin Policy）限制了一个域名下的网页
//	向另一个域名发送 AJAX 请求。CORS 是一种机制，通过特定的 HTTP 头
//	告诉浏览器"允许来自某某域名的请求"。
//
//	例如：前端运行在 http://localhost:5173（Vite 开发服务器），
//	后端运行在 http://localhost:8080，这是两个不同的"源"
//	（协议+域名+端口任一不同就算不同源），浏览器会阻止跨域请求。
//
//	CORS 中间件的作用就是在响应中加上以下头：
//
//	Access-Control-Allow-Origin: http://localhost:5173
//	  → 告诉浏览器：允许来自这个地址的跨域请求
//
//	Access-Control-Allow-Methods: GET, POST, PUT, DELETE, OPTIONS
//	  → 告诉浏览器：允许这些 HTTP 方法
//
//	Access-Control-Allow-Headers: Content-Type, Authorization
//	  → 告诉浏览器：允许携带这些请求头
//
//	Access-Control-Allow-Credentials: true
//	  → 告诉浏览器：允许携带 Cookie/认证信息
//
// OPTIONS 预检请求：
//
//	浏览器在发送某些跨域请求之前（如带 Content-Type: application/json 的 POST），
//	会先发一个 OPTIONS 请求来"探路"，询问服务器是否允许跨域。
//	这个中间件收到 OPTIONS 请求后直接返回 204（No Content），不做其他处理。
func CORS() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 允许的前端地址（Vite 默认开发服务器端口）
		// 生产环境应该改为实际的前端部署地址
		c.Header("Access-Control-Allow-Origin", "http://localhost:5173")
		// 允许的 HTTP 方法
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		// 允许的请求头（前端可以发送这些头部信息）
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")
		// 允许携带 Cookie 等认证凭据
		c.Header("Access-Control-Allow-Credentials", "true")

		// 处理 OPTIONS 预检请求
		// OPTIONS 请求不需要做业务处理，直接返回 204
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent) // 204 No Content
			return
		}
		// 非 OPTIONS 请求继续处理
		c.Next()
	}
}
