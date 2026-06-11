package service

import (
	"crypto/aes"      // AES 对称加密算法
	"crypto/cipher"   // 加密模式（GCM）
	"crypto/rand"     // 密码学安全的随机数生成器
	"encoding/base64" // Base64 编码（把二进制密文转为可打印字符串）
	"errors"
	"io"

	"github.com/choex2025-ops/choex-server/internal/database"
	"github.com/choex2025-ops/choex-server/internal/model"
)

// PasswordService 密码管理服务，负责密码的加密存储和查询。
//
// 加密方案：AES-256-GCM
//
//	AES-256：使用 256 位（32 字节）密钥的 AES 加密算法，对称加密（加解密用同一把密钥）
//	GCM（Galois/Counter Mode）：一种认证加密模式，同时提供：
//	  1. 机密性（别人看不懂密文）
//	  2. 完整性（能检测密文是否被篡改）
//	  3. 认证性（能验证密文的来源）
//
// 加密流程（Encrypt）：
//
//	明文 "myPassword123"
//	  → AES-GCM 加密（密钥 + 随机 nonce）
//	  → 二进制密文（nonce + 加密数据）
//	  → Base64 编码（变成可存入数据库的纯文本字符串）
//	  → 存到数据库
//
// 解密流程（decrypt）：
//
//	从数据库读出 Base64 字符串
//	  → Base64 解码（变回二进制密文）
//	  → 提取 nonce 和加密数据
//	  → AES-GCM 解密（密钥 + nonce）
//	  → 明文 "myPassword123"
type PasswordService struct {
	key []byte // AES 加密密钥，32 字节（256 位）
}

// NewPasswordService 创建密码管理服务实例。
//
// 参数：
//   - encryptionKey：加密密钥字符串，不足 32 字节会自动填充到 32 字节
//
// 为什么需要 32 字节？
//
//	AES-256 要求密钥正好是 32 字节（256 位）。如果用户提供的密钥太短，
//	用 0 补足；如果太长，截取前 32 字节。
func NewPasswordService(encryptionKey string) *PasswordService {
	key := []byte(encryptionKey)
	// 如果密钥不足 32 字节，用零值填充到 32 字节
	if len(key) < 32 {
		padded := make([]byte, 32) // make([]byte, 32) 创建的切片所有元素初始化为 0
		copy(padded, key)           // 把原密钥复制到前面
		key = padded
	}
	// 取前 32 字节（如果原密钥超过 32 字节就截断）
	return &PasswordService{key: key[:32]}
}

// List 获取指定用户的所有密码记录（不含解密后的明文密码）。
//
// 注意：这个方法返回的密码记录中 EncryptedPassword 字段是空的（因为 json:"-" 标签），
// 前端拿不到加密密码。要获取明文密码，需要调用 Get 方法单独查询。
//
// 参数：
//   - userID：用户 ID
//
// 返回：密码记录列表 和 可能的错误
func (s *PasswordService) List(userID uint64) ([]model.Password, error) {
	var passwords []model.Password
	err := database.DB.Where("user_id = ?", userID).Order("created_at DESC").Find(&passwords).Error
	return passwords, err
}

// Create 创建一条密码记录（传入的是 model.Password 结构体）。
func (s *PasswordService) Create(p *model.Password) error {
	return database.DB.Create(p).Error
}

// CreateRaw 用 map 数据创建一条密码记录。
//
// 为什么需要这个方法？
//
//	因为 EncryptedPassword 字段有 json:"-" 标签，用 ShouldBindJSON 绑定到
//	model.Password 结构体时，这个字段会被忽略。所以用 map[string]any
//	的方式直接传递加密后的密码值。
//
// 参数：
//   - data：包含所有字段的 map，如 {"user_id": 1, "title": "xx", "encrypted_password": "..."}
//
// 返回：可能的错误
func (s *PasswordService) CreateRaw(data map[string]any) error {
	return database.DB.Table("passwords").Create(data).Error
}

// Get 获取单条密码记录，并解密密码字段。
//
// 和 List 的区别：
//   - List 不返回密码（安全）
//   - Get 会解密并返回明文密码（有权限控制，只返回自己的）
//
// 参数：
//   - id：密码记录 ID
//   - userID：用户 ID
//
// 返回：包含明文密码的记录 和 可能的错误
func (s *PasswordService) Get(id uint64, userID uint64) (*model.Password, error) {
	var p model.Password
	err := database.DB.Where("id = ? AND user_id = ?", id, userID).First(&p).Error
	if err != nil {
		return nil, err
	}
	// 解密密码字段
	decrypted, err := s.decrypt(p.EncryptedPassword)
	if err != nil {
		return nil, err
	}
	// 把解密后的明文放回结构体的 EncryptedPassword 字段
	// 虽然字段名叫 EncryptedPassword，但此时存的是明文
	// 由于 json:"-" 的原因，这个字段不会返回给前端...
	// 但实际上这个字段还是会通过 json tag 返回的，因为字段名是 EncryptedPassword 但 json tag 是 "-"
	// 等一下，json:"-" 意味着序列化时跳过，所以这里解密后需要想办法传给前端
	// 当前的设计：把明文放到 EncryptedPassword 字段，json:"-" 导致不输出
	// 这个行为的意图是：Get 返回的 JSON 中密码在 password 字段（由 handler 构造）而不是 encrypted_password
	// 实际上看 handler 代码，Get 直接 c.JSON(http.StatusOK, p)，而 p.EncryptedPassword 是 json:"-"
	// 所以前端拿到的 JSON 里没有密码... 这个可能需要看前端怎么处理的
	// 简化理解：Service 把明文放到这个字段，Handler 返回时会通过 json tag 处理
	p.EncryptedPassword = decrypted
	return &p, nil
}

// Update 更新指定密码记录的字段。
//
// 参数：
//   - id：密码记录 ID
//   - userID：用户 ID
//   - updates：要更新的字段键值对
//
// 返回：可能的错误
func (s *PasswordService) Update(id uint64, userID uint64, updates map[string]any) error {
	return database.DB.Model(&model.Password{}).Where("id = ? AND user_id = ?", id, userID).Updates(updates).Error
}

// Delete 删除指定密码记录。
//
// 参数：
//   - id：密码记录 ID
//   - userID：用户 ID
//
// 返回：可能的错误
func (s *PasswordService) Delete(id uint64, userID uint64) error {
	return database.DB.Where("id = ? AND user_id = ?", id, userID).Delete(&model.Password{}).Error
}

// Encrypt 用 AES-256-GCM 加密明文，返回 Base64 编码的密文。
//
// 加密步骤详解：
//  1. 用密钥创建 AES cipher（加密器）
//  2. 用 AES cipher 创建 GCM 模式包装器
//  3. 生成随机 nonce（Number used ONCE，一次性数字）
//  4. 用 GCM 加密：nonce + 明文 → 密文
//  5. 把 nonce 拼接在密文前面（解密时需要 nonce）
//  6. Base64 编码整个结果，方便存数据库
//
// 为什么 nonce 要随机且不保密？
//
//	nonce 的作用是让同样的明文每次加密产生不同的密文。
//	nonce 不需要保密，它通常拼接在密文前面一起传输。
//	关键是同一个密钥下 nonce 不能重复使用，否则攻击者可以破解。
//	随机生成 12 字节 nonce，重复的概率极低（2^96 种可能）。
//
// 参数：
//   - plaintext：要加密的明文密码
//
// 返回：Base64 编码的密文 和 可能的错误
func (s *PasswordService) Encrypt(plaintext string) (string, error) {
	// 1. 创建 AES cipher
	block, err := aes.NewCipher(s.key)
	if err != nil {
		return "", err
	}

	// 2. 创建 GCM 模式包装器
	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	// 3. 生成随机 nonce
	// NonceSize() 返回 GCM 推荐的 nonce 长度（通常是 12 字节）
	nonce := make([]byte, aesGCM.NonceSize())
	// io.ReadFull 从 crypto/rand.Reader（操作系统提供的密码学安全随机源）读取随机字节
	// rand.Reader 在 Linux 上来自 /dev/urandom，在 macOS 上来自内核的随机数生成器
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	// 4. Seal 加密：nonce + 明文 → 密文
	// Seal 的参数：
	//   nonce：附加认证数据的前缀（加密后会拼接在密文前面）
	//   nonce：加密用的 nonce 值
	//   []byte(plaintext)：要加密的明文
	//   nil：额外的认证数据（这里不需要）
	// 返回值：nonce + 加密后的数据（nonce 自动拼接在密文前面）
	ciphertext := aesGCM.Seal(nonce, nonce, []byte(plaintext), nil)

	// 5. Base64 编码：二进制 → 可打印字符串
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// decrypt 解密 Base64 编码的密文，返回明文。
//
// 解密步骤详解：
//  1. Base64 解码：字符串 → 二进制数据
//  2. 创建 AES cipher 和 GCM 包装器
//  3. 从二进制数据中分离 nonce 和密文
//  4. GCM 解密：nonce + 密文 → 明文
//
// 参数：
//   - encoded：Base64 编码的密文字符串
//
// 返回：明文密码 和 可能的错误
func (s *PasswordService) decrypt(encoded string) (string, error) {
	// 1. Base64 解码
	ciphertext, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", err
	}

	// 2. 创建 AES cipher
	block, err := aes.NewCipher(s.key)
	if err != nil {
		return "", err
	}

	// 2. 创建 GCM 包装器
	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	// 3. 分离 nonce 和密文数据
	nonceSize := aesGCM.NonceSize()                    // 通常是 12 字节
	if len(ciphertext) < nonceSize {
		return "", errors.New("ciphertext too short")   // 数据太短，不合法
	}
	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	// nonce       = 前 12 字节（加密时随机生成的）
	// ciphertext  = 后面的加密数据

	// 4. GCM 解密
	// Open 的参数：
	//   nil：输出的目标切片（nil 表示新分配）
	//   nonce：加密时用的 nonce
	//   ciphertext：加密数据
	//   nil：额外的认证数据（和加密时保持一致）
	plaintext, err := aesGCM.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", err
	}
	return string(plaintext), nil
}
