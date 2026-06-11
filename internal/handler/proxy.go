package handler

import (
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"
)

func ProxyHandler(c *gin.Context) {
	targetURL := c.Query("url")
	if targetURL == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "url parameter required"})
		return
	}

	parsed, err := url.ParseRequestURI(targetURL)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid url"})
		return
	}

	resp, err := http.Get(targetURL)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "failed to fetch url"})
		return
	}
	defer resp.Body.Close()

	contentType := resp.Header.Get("Content-Type")

	// Pass through headers we care about
	for _, h := range []string{"Content-Type", "Content-Length"} {
		if v := resp.Header.Get(h); v != "" {
			c.Header(h, v)
		}
	}

	body, _ := io.ReadAll(resp.Body)

	// Inject <base> tag into HTML so relative URLs resolve correctly
	if strings.Contains(contentType, "text/html") {
		baseTag := "<base href=\"" + targetURL + "\">"
		body = []byte(strings.Replace(string(body), "<head>", "<head>"+baseTag, 1))
		c.Header("Content-Length", "")
	}

	baseURL := parsed.Scheme + "://" + parsed.Host
	c.Header("X-Proxy-Base", baseURL)
	c.Data(resp.StatusCode, contentType, body)
}
