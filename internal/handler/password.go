package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/choex2025-ops/choex-server/internal/model"
	"github.com/choex2025-ops/choex-server/internal/service"
)

type PasswordHandler struct {
	svc *service.PasswordService
}

func NewPasswordHandler(svc *service.PasswordService) *PasswordHandler {
	return &PasswordHandler{svc: svc}
}

type passwordBody struct {
	Title    string `json:"title"`
	URL      string `json:"url"`
	Username string `json:"username"`
	Password string `json:"password"`
	Note     string `json:"note"`
	Category string `json:"category"`
}

func (h *PasswordHandler) List(c *gin.Context) {
	userID := c.GetUint64("user_id")
	passwords, err := h.svc.List(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if passwords == nil {
		passwords = []model.Password{}
	}
	c.JSON(http.StatusOK, passwords)
}

func (h *PasswordHandler) Create(c *gin.Context) {
	var body passwordBody
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	encrypted, err := h.svc.Encrypt(body.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "encryption failed"})
		return
	}

	p := map[string]any{
		"user_id":            c.GetUint64("user_id"),
		"title":              body.Title,
		"url":                body.URL,
		"username":           body.Username,
		"encrypted_password": encrypted,
		"note":               body.Note,
		"category":           body.Category,
	}

	if err := h.svc.CreateRaw(p); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"title": body.Title, "username": body.Username, "url": body.URL})
}

func (h *PasswordHandler) Get(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	userID := c.GetUint64("user_id")
	p, err := h.svc.Get(id, userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	c.JSON(http.StatusOK, p)
}

func (h *PasswordHandler) Update(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	userID := c.GetUint64("user_id")

	var body passwordBody
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	updates := map[string]any{
		"title":    body.Title,
		"url":      body.URL,
		"username": body.Username,
		"note":     body.Note,
		"category": body.Category,
	}
	if body.Password != "" {
		encrypted, err := h.svc.Encrypt(body.Password)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "encryption failed"})
			return
		}
		updates["encrypted_password"] = encrypted
	}

	if err := h.svc.Update(id, userID, updates); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "updated"})
}

func (h *PasswordHandler) Delete(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	userID := c.GetUint64("user_id")
	if err := h.svc.Delete(id, userID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "deleted"})
}
