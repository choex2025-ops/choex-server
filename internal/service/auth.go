// Package service 包含所有业务逻辑层的代码。
//
// 在 MVC/三层架构中，service 层的作用：
//   handler（控制器）→ 接收请求、参数校验、返回响应
//   service （服务层） → 业务逻辑、数据处理、算法实现
//   model  （模型层） → 数据结构定义
//
// 把业务逻辑放在 service 层的好处：
//   1. handler 保持简洁，只做"接请求 → 调 service → 返回应"
//   2. 业务逻辑可复用（多个 handler 可以调用同一个 service）
//   3. 方便单元测试（不依赖 HTTP 请求上下文即可测试业务逻辑）
package service

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5" // JWT（JSON Web Token）库
	"golang.org/x/crypto/bcrypt"    // bcrypt 密码哈希库

	"github.com/choex2025-ops/choex-server/internal/config"
)

// ---- 预定义的业务错误 ----
// 使用 errors.New 创建简单的错误常量，方便在 handler 层判断错误类型
var (
	ErrInvalidCredentials = errors.New("invalid credentials") // 登录凭据无效（邮箱或密码错误）
	ErrUserExists         = errors.New("user already exists") // 用户已存在（注册时邮箱重复）
)

// AuthService 认证服务，负责用户的注册、登录、令牌管理。
//
// 认证流程（简化版）：
//
//	注册：用户提交邮箱+密码 → bcrypt 加密密码 → 存入数据库 → 返回 JWT 令牌
//	登录：用户提交邮箱+密码 → 查数据库 → bcrypt 验证密码 → 返回 JWT 令牌
//	鉴权：请求带 JWT 令牌 → 中间件解析令牌 → 提取 user_id 放入上下文
type AuthService struct {
	cfg *config.Config // 配置（主要用 JWTSecret 来签发令牌）
}

// NewAuthService 创建认证服务实例。
//
// 参数：
//   - cfg：应用配置
//
// 返回：*AuthService 实例
func NewAuthService(cfg *config.Config) *AuthService {
	return &AuthService{cfg: cfg}
}

// Claims 是 JWT 令牌的载荷（Payload），包含用户身份信息。
//
// JWT 令牌的结构（用 . 分隔的三段 Base64）：
//
//	Header.Payload.Signature
//	eyJhbG... . eyJ1c2Vy... . SflKxw...
//
// Header：令牌类型和签名算法（HS256）
// Payload：Claims 里的数据（user_id, email, 过期时间等）
// Signature：签名 = HMAC-SHA256(Header + "." + Payload, secret)
//           用于验证令牌没有被篡改（只有知道 secret 的人才能生成/验证签名）
type Claims struct {
	UserID uint64 `json:"user_id"` // 用户 ID
	Email  string `json:"email"`   // 用户邮箱
	// jwt.RegisteredClaims 是 JWT 标准字段的集合，包含：
	//   - ExpiresAt：令牌过期时间（超过这个时间令牌无效）
	//   - IssuedAt：令牌签发时间
	//   - Issuer：签发者
	//   - Subject：主题
	jwt.RegisteredClaims
}

// HashPassword 用 bcrypt 加密明文密码。
//
// bcrypt 是什么？
//
//	一种专门为密码存储设计的哈希算法，特点是：
//	1. 自带盐值（salt），每次加密结果不同，即使相同密码也产生不同哈希
//	2. 故意设计得很慢，增加暴力破解的难度
//	3. 不可逆——无法从哈希值反推出原密码
//
// bcrypt.DefaultCost = 10，表示进行 2^10 = 1024 轮哈希迭代
// 参数：
//   - password：用户输入的明文密码
//
// 返回：加密后的哈希字符串 和 可能的错误
//
// 示例输出：$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy
//
//	$2a$ → bcrypt 版本
//	10$  → cost 参数
//	后面 → 22 字符盐值 + 31 字符哈希值
func (s *AuthService) HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

// CheckPassword 验证明文密码是否与 bcrypt 哈希值匹配。
//
// 这个验证过程是安全的，因为：
//   1. 哈希值中已经包含了盐值（它就是哈希值的前 29 个字符）
//   2. bcrypt.CompareHashAndPassword 会提取盐值，用同样的参数计算哈希
//   3. 比较计算结果和存储的哈希值是否一致
//
// 参数：
//   - password：用户输入的明文密码
//   - hash：数据库里存储的 bcrypt 哈希值
//
// 返回：true 表示密码正确，false 表示密码错误
func (s *AuthService) CheckPassword(password, hash string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}

// GenerateToken 为用户生成 JWT 访问令牌。
//
// 令牌包含的信息：
//   - user_id：用户 ID，后续请求通过它来识别"是谁在操作"
//   - email：用户邮箱
//   - exp：过期时间（签发后 24 小时）
//   - iat：签发时间
//
// 签名算法：HS256（HMAC-SHA256），使用配置中的 JWTSecret 作为密钥
//
// 为什么用 JWT 而不是 Session？
//   JWT 是无状态的——服务端不需要存储会话信息，只需要验证签名即可。
//   适合微服务架构（多个服务共享同一个 secret 就能验证令牌）。
//
// 参数：
//   - userID：用户 ID
//   - email：用户邮箱
//
// 返回：JWT 令牌字符串 和 可能的错误
func (s *AuthService) GenerateToken(userID uint64, email string) (string, error) {
	claims := &Claims{
		UserID: userID,
		Email:  email,
		RegisteredClaims: jwt.RegisteredClaims{
			// 过期时间：当前时间 + 24 小时
			// jwt.NewNumericDate 把 time.Time 转为 JWT 标准的时间格式
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			// 签发时间：当前时间
			IssuedAt: jwt.NewNumericDate(time.Now()),
		},
	}
	// jwt.NewWithClaims 创建一个未签名的令牌对象
	// SigningMethodHS256：使用 HMAC-SHA256 签名算法
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	// SignedString 用密钥对令牌签名，返回完整的 JWT 字符串
	return token.SignedString([]byte(s.cfg.JWTSecret))
}

// ParseToken 解析并验证 JWT 令牌，提取其中的用户信息。
//
// 验证过程：
//   1. 解析令牌的三段结构（Header.Payload.Signature）
//   2. 用同样的 secret 验证签名是否匹配
//   3. 检查是否过期（exp 字段）
//   4. 如果全部通过，提取 Claims 中的数据
//
// 参数：
//   - tokenString：HTTP 请求头中的 JWT 字符串（去掉 "Bearer " 前缀后）
//
// 返回：*Claims（用户信息） 和 可能的错误
func (s *AuthService) ParseToken(tokenString string) (*Claims, error) {
	// jwt.ParseWithClaims 做三件事：
	//   1. 解析令牌字符串
	//   2. 用 keyFunc 返回的密钥验证签名
	//   3. 把 Payload 部分反序列化到 Claims 结构体
	token, err := jwt.ParseWithClaims(tokenString, &Claims{},
		func(token *jwt.Token) (any, error) {
			// keyFunc：返回验证签名用的密钥
			// 如果有人篡改了 Payload，签名就对不上，验证失败
			return []byte(s.cfg.JWTSecret), nil
		},
	)
	if err != nil || !token.Valid {
		return nil, ErrInvalidCredentials
	}

	// 类型断言：把 token.Claims 这个接口类型转为具体的 *Claims 类型
	claims, ok := token.Claims.(*Claims)
	if !ok {
		return nil, ErrInvalidCredentials
	}
	return claims, nil
}
