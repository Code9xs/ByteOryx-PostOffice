package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/byteoryx/postoffice/internal/config"
	"github.com/byteoryx/postoffice/internal/domain/model"
	"github.com/byteoryx/postoffice/internal/domain/repository"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/argon2"
)

type AuthService struct {
	userRepo repository.UserRepository
	cfg      *config.JWTConfig
}

func NewAuthService(userRepo repository.UserRepository, cfg *config.JWTConfig) *AuthService {
	return &AuthService{userRepo: userRepo, cfg: cfg}
}

func (s *AuthService) Register(ctx context.Context, email, password, displayName string) (*model.User, error) {
	hash := hashPassword(password)
	user := &model.User{
		ID:           uuid.New(),
		Email:        email,
		PasswordHash: hash,
		DisplayName:  displayName,
		Role:         "user",
		IsActive:     true,
		QuotaBytes:   1073741824, // 1GB
	}
	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, fmt.Errorf("create user: %w", err)
	}
	return user, nil
}

func (s *AuthService) Login(ctx context.Context, email, password string) (string, *model.User, error) {
	user, err := s.userRepo.GetByEmail(ctx, email)
	if err != nil {
		return "", nil, fmt.Errorf("invalid credentials")
	}
	if !user.IsActive {
		return "", nil, fmt.Errorf("account disabled")
	}
	if !verifyPassword(password, user.PasswordHash) {
		return "", nil, fmt.Errorf("invalid credentials")
	}

	token, err := s.generateToken(user)
	if err != nil {
		return "", nil, err
	}
	return token, user, nil
}

func (s *AuthService) ValidateToken(tokenStr string) (*jwt.MapClaims, error) {
	token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method")
		}
		return []byte(s.cfg.Secret), nil
	})
	if err != nil {
		return nil, err
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}
	return &claims, nil
}

func (s *AuthService) generateToken(user *model.User) (string, error) {
	claims := jwt.MapClaims{
		"sub":  user.ID.String(),
		"email": user.Email,
		"role": user.Role,
		"exp":  time.Now().Add(time.Duration(s.cfg.ExpireHour) * time.Hour).Unix(),
		"iat":  time.Now().Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.cfg.Secret))
}

func hashPassword(password string) string {
	salt := make([]byte, 16)
	rand.Read(salt)
	hash := argon2.IDKey([]byte(password), salt, 1, 64*1024, 4, 32)
	return fmt.Sprintf("%s$%s", hex.EncodeToString(salt), hex.EncodeToString(hash))
}

func verifyPassword(password, stored string) bool {
	parts := splitN(stored, "$", 2)
	if len(parts) != 2 {
		return false
	}
	salt, _ := hex.DecodeString(parts[0])
	expectedHash, _ := hex.DecodeString(parts[1])
	hash := argon2.IDKey([]byte(password), salt, 1, 64*1024, 4, 32)
	return constantTimeEqual(hash, expectedHash)
}

func splitN(s, sep string, n int) []string {
	result := make([]string, 0, n)
	for i := 0; i < n-1; i++ {
		idx := indexOf(s, sep)
		if idx < 0 {
			break
		}
		result = append(result, s[:idx])
		s = s[idx+len(sep):]
	}
	result = append(result, s)
	return result
}

func indexOf(s, sub string) int {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}

func constantTimeEqual(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	var result byte
	for i := range a {
		result |= a[i] ^ b[i]
	}
	return result == 0
}
