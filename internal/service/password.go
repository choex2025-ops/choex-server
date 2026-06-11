package service

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"io"

	"github.com/choex2025-ops/choex-server/internal/database"
	"github.com/choex2025-ops/choex-server/internal/model"
)

type PasswordService struct {
	key []byte
}

func NewPasswordService(encryptionKey string) *PasswordService {
	key := []byte(encryptionKey)
	if len(key) < 32 {
		padded := make([]byte, 32)
		copy(padded, key)
		key = padded
	}
	return &PasswordService{key: key[:32]}
}

func (s *PasswordService) List(userID uint64) ([]model.Password, error) {
	var passwords []model.Password
	err := database.DB.Where("user_id = ?", userID).Order("created_at DESC").Find(&passwords).Error
	return passwords, err
}

func (s *PasswordService) Create(p *model.Password) error {
	return database.DB.Create(p).Error
}

func (s *PasswordService) Get(id uint64, userID uint64) (*model.Password, error) {
	var p model.Password
	err := database.DB.Where("id = ? AND user_id = ?", id, userID).First(&p).Error
	if err != nil {
		return nil, err
	}
	decrypted, err := s.decrypt(p.EncryptedPassword)
	if err != nil {
		return nil, err
	}
	p.EncryptedPassword = decrypted
	return &p, nil
}

func (s *PasswordService) Update(id uint64, userID uint64, updates map[string]any) error {
	return database.DB.Model(&model.Password{}).Where("id = ? AND user_id = ?", id, userID).Updates(updates).Error
}

func (s *PasswordService) Delete(id uint64, userID uint64) error {
	return database.DB.Where("id = ? AND user_id = ?", id, userID).Delete(&model.Password{}).Error
}

func (s *PasswordService) Encrypt(plaintext string) (string, error) {
	block, err := aes.NewCipher(s.key)
	if err != nil {
		return "", err
	}
	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, aesGCM.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	ciphertext := aesGCM.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

func (s *PasswordService) decrypt(encoded string) (string, error) {
	ciphertext, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", err
	}
	block, err := aes.NewCipher(s.key)
	if err != nil {
		return "", err
	}
	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonceSize := aesGCM.NonceSize()
	if len(ciphertext) < nonceSize {
		return "", errors.New("ciphertext too short")
	}
	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := aesGCM.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", err
	}
	return string(plaintext), nil
}
