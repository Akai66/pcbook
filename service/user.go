package service

import (
	"fmt"
	"golang.org/x/crypto/bcrypt"
)

type User struct {
	Username       string
	HashedPassword string //加密后的密码
	Role           string
}

// NewUser 创建一个用户对象
func NewUser(username, password, role string) (*User, error) {
	//对明文字符串密码加密
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("cannot hash password: %w", err)
	}
	user := &User{
		Username:       username,
		HashedPassword: string(hashedPassword),
		Role:           role,
	}
	return user, nil
}

// IsCorrectPassword 判断密码是否正确
func (user *User) IsCorrectPassword(password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(user.HashedPassword), []byte(password))
	return err == nil
}

// Clone 复制User对象
func (user *User) Clone() *User {
	return &User{
		Username:       user.Username,
		HashedPassword: user.HashedPassword,
		Role:           user.Role,
	}
}
