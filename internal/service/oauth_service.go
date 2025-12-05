// FILE: internal/service/oauth_service.go
package service

import (
	"ai-notetaking-be/internal/dto"
	"ai-notetaking-be/internal/entity"
	"ai-notetaking-be/internal/repository"
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

type IOAuthService interface {
	GetLoginURL(provider string) (string, error)
	HandleCallback(ctx context.Context, provider string, code string) (*dto.LoginResponse, error)
}

type oauthService struct {
	userRepo   repository.IUserRepository
	googleConf *oauth2.Config
}

func NewOAuthService(userRepo repository.IUserRepository) IOAuthService {
	// Initialize Google OAuth Config
	// In production, these should be strictly loaded from ENV
	conf := &oauth2.Config{
		ClientID:     os.Getenv("GOOGLE_CLIENT_ID"),
		ClientSecret: os.Getenv("GOOGLE_CLIENT_SECRET"),
		RedirectURL:  os.Getenv("GOOGLE_REDIRECT_URL"), // e.g., http://localhost:3000/api/auth/google/callback
		Scopes: []string{
			"https://www.googleapis.com/auth/userinfo.email",
			"https://www.googleapis.com/auth/userinfo.profile",
		},
		Endpoint: google.Endpoint,
	}

	return &oauthService{
		userRepo:   userRepo,
		googleConf: conf,
	}
}

func (s *oauthService) GetLoginURL(provider string) (string, error) {
	if provider != "google" {
		return "", errors.New("unsupported provider")
	}
	
	// Generate random state
	b := make([]byte, 16)
	rand.Read(b)
	state := base64.URLEncoding.EncodeToString(b)
	
	// Return the URL
	return s.googleConf.AuthCodeURL(state), nil
}

func (s *oauthService) HandleCallback(ctx context.Context, provider string, code string) (*dto.LoginResponse, error) {
	if provider != "google" {
		return nil, errors.New("unsupported provider")
	}

	// 1. Exchange Code for Token
	token, err := s.googleConf.Exchange(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("code exchange failed: %v", err)
	}

	// 2. Fetch User Info
	resp, err := http.Get("https://www.googleapis.com/oauth2/v2/userinfo?access_token=" + token.AccessToken)
	if err != nil {
		return nil, fmt.Errorf("failed getting user info: %v", err)
	}
	defer resp.Body.Close()

	content, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed reading response: %v", err)
	}

	var googleUser struct {
		ID            string `json:"id"`
		Email         string `json:"email"`
		VerifiedEmail bool   `json:"verified_email"`
		Name          string `json:"name"`
		Picture       string `json:"picture"`
	}

	if err := json.Unmarshal(content, &googleUser); err != nil {
		return nil, err
	}

	// 3. Find or Create User in DB
	user, err := s.userRepo.GetByEmail(ctx, googleUser.Email)
	if err != nil {
		// Database error
		return nil, err
	}

	if user == nil {
		// Create new user
		newUser := &entity.User{
			Id:            uuid.New(),
			Email:         googleUser.Email,
			FullName:      googleUser.Name,
			PasswordHash:  nil, // No password for OAuth
			Role:          entity.UserRoleUser,
			Status:        entity.UserStatusActive,
			EmailVerified: true,
			CreatedAt:     time.Now(),
			UpdatedAt:     time.Now(),
		}
		if err := s.userRepo.Create(ctx, newUser); err != nil {
			return nil, err
		}
		user = newUser
	}

	// 4. Generate JWT
	claims := jwt.MapClaims{
		"user_id": user.Id.String(),
		"role":    user.Role,
		"exp":     time.Now().Add(time.Hour * 24).Unix(),
	}
	jwtToken := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	secret := os.Getenv("JWT_SECRET")
	signedToken, err := jwtToken.SignedString([]byte(secret))
	if err != nil {
		return nil, err
	}

	return &dto.LoginResponse{
		AccessToken: signedToken,
		User: dto.UserDTO{
			Id:       user.Id,
			Email:    user.Email,
			FullName: user.FullName,
			Role:     string(user.Role),
		},
	}, nil
}