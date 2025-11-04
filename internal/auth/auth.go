package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"sigs.k8s.io/controller-runtime/pkg/client"

	llmcloudv1alpha1 "github.com/rusik69/llmcloud-operator/api/v1alpha1"
)

var jwtSecret []byte

// InitJWTSecret initializes the JWT secret (should be called once at startup)
func InitJWTSecret() error {
	jwtSecret = make([]byte, 32)
	_, err := rand.Read(jwtSecret)
	return err
}

// Claims represents the JWT claims
type Claims struct {
	Username string   `json:"username"`
	IsAdmin  bool     `json:"isAdmin"`
	Projects []string `json:"projects"`
	jwt.RegisteredClaims
}

// HashPassword hashes a password using bcrypt
func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

// CheckPasswordHash compares a password with its hash
func CheckPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

// GeneratePassword generates a random password
func GeneratePassword(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(bytes)[:length], nil
}

// GenerateJWT generates a JWT token for a user
func GenerateJWT(user *llmcloudv1alpha1.User) (string, error) {
	if jwtSecret == nil {
		return "", fmt.Errorf("JWT secret not initialized")
	}

	claims := Claims{
		Username: user.Spec.Username,
		IsAdmin:  user.Spec.IsAdmin,
		Projects: user.Spec.Projects,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtSecret)
}

// ValidateJWT validates a JWT token and returns the claims
func ValidateJWT(tokenString string) (*Claims, error) {
	if jwtSecret == nil {
		return nil, fmt.Errorf("JWT secret not initialized")
	}

	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return jwtSecret, nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, fmt.Errorf("invalid token")
}

// AuthenticateUser authenticates a user by username and password
func AuthenticateUser(ctx context.Context, k8sClient client.Client, username, password string) (*llmcloudv1alpha1.User, error) {
	// List all users (they are cluster-scoped)
	userList := &llmcloudv1alpha1.UserList{}
	if err := k8sClient.List(ctx, userList); err != nil {
		return nil, err
	}

	// Find user by username
	for i := range userList.Items {
		user := &userList.Items[i]
		if user.Spec.Username == username {
			if user.Spec.Disabled {
				return nil, fmt.Errorf("user account is disabled")
			}
			if CheckPasswordHash(password, user.Spec.PasswordHash) {
				return user, nil
			}
			return nil, fmt.Errorf("invalid password")
		}
	}

	return nil, fmt.Errorf("user not found")
}

// HasProjectAccess checks if a user has access to a project
func HasProjectAccess(claims *Claims, projectName string) bool {
	if claims.IsAdmin {
		return true
	}
	for _, p := range claims.Projects {
		if p == projectName {
			return true
		}
	}
	return false
}
