package handler

import (
	"io"
	"net/http"
	"net/url"

	"github.com/gin-gonic/gin"
)

func ProxyHandler(c *gin.Context) {
	targetURL := c.Query("url")
	if targetURL == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "url parameter required"})
		return
	}

	if _, err := url.ParseRequestURI(targetURL); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid url"})
		return
	}

	resp, err := http.Get(targetURL)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "failed to fetch url"})
		return
	}
	defer resp.Body.Close()

	for k, v := range resp.Header {
		for _, vv := range v {
			c.Header(k, vv)
		}
	}
	c.Status(resp.StatusCode)
	io.Copy(c.Writer, resp.Body)
}
