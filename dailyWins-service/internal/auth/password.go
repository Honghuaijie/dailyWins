package auth

import (
	"fmt"
	"golang.org/x/crypto/bcrypt"
	"unicode/utf8"
)

// bcrypt 自带随机盐；限制字节长度，避免算法的 72 字节截断边界。
func HashPassword(password string) (string, error) {
	if utf8.RuneCountInString(password) < 6 || len(password) > 72 {
		return "", fmt.Errorf("密码至少 6 个字符，且不超过 72 字节")
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	return string(hash), err
}

func CheckPassword(hash, password string) bool {
	return len(password) <= 72 && bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}
