package service

import (
	"fmt"
	"github.com/dgrijalva/jwt-go"
	"time"
)

//JWTManager 为用户生成和验证访问令牌的json web token管理器
type JWTManager struct {
	secretKey     string        //用于签署和验证访问令牌的密钥
	tokenDuration time.Duration //令牌的有效期限
}

// UserClaims JWT必须的，记录拥有它的用户的一些信息
type UserClaims struct {
	jwt.StandardClaims        //jwt的标准声明字段
	Username           string `json:"username"`
	Role               string `json:"role"`
}

func NewJWTManager(secretKey string, tokenDuration time.Duration) *JWTManager {
	return &JWTManager{
		secretKey:     secretKey,
		tokenDuration: tokenDuration,
	}
}

// Generate 为特定的用户生成并签名新的访问token
func (manager *JWTManager) Generate(user *User) (string, error) {
	//构造claims
	claims := UserClaims{
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: time.Now().Add(manager.tokenDuration).Unix(),
		},
		Username: user.Username,
		Role:     user.Role,
	}
	//创建jwt token对象
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	//使用私有密钥签名生成的token，防止token被伪造
	return token.SignedString([]byte(manager.secretKey))
}

// Verify 验证访问的token是否为有效的，有效时，返回token中的claims信息
func (manager *JWTManager) Verify(accessToken string) (*UserClaims, error) {
	//解析json web token
	token, err := jwt.ParseWithClaims(
		accessToken,
		&UserClaims{},
		func(token *jwt.Token) (interface{}, error) {
			//该方法负责，自定义一些验证逻辑，最后返回签名的密钥
			_, ok := token.Method.(*jwt.SigningMethodHMAC)
			if !ok {
				return nil, fmt.Errorf("unexpected token signing method")
			}
			return []byte(manager.secretKey), nil
		},
	)
	if err != nil {
		return nil, fmt.Errorf("invalid token: %w", err)
	}

	//token解析成功后，获取Claims字段并断言，转换为*UserClaims
	claims, ok := token.Claims.(*UserClaims)
	if !ok {
		return nil, fmt.Errorf("invalid token claims")
	}
	return claims, nil
}
