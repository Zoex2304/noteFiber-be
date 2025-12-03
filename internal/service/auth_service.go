// FILE: internal/service/auth_service.go
package service

import (
	"ai-notetaking-be/internal/dto"
	"ai-notetaking-be/internal/entity"
	"ai-notetaking-be/internal/repository"
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type IAuthService interface {
	Register(ctx context.Context, req *dto.RegisterRequest) (*dto.RegisterResponse, error)
	Login(ctx context.Context, req *dto.LoginRequest) (*dto.LoginResponse, error)
	ForgotPassword(ctx context.Context, req *dto.ForgotPasswordRequest) error
	ResetPassword(ctx context.Context, req *dto.ResetPasswordRequest) error
	VerifyEmail(ctx context.Context, req *dto.VerifyEmailRequest) error // New
}

type authService struct {
	userRepo repository.IUserRepository
}

func NewAuthService(userRepo repository.IUserRepository) IAuthService {
	return &authService{userRepo: userRepo}
}

// Helper to generate secure 6-digit OTP
func generateOTP() (string, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(1000000))
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%06d", n), nil
}

func (s *authService) Register(ctx context.Context, req *dto.RegisterRequest) (*dto.RegisterResponse, error) {
	// 1. Check for existing user
	existing, _ := s.userRepo.GetByEmail(ctx, req.Email)
	if existing != nil {
		// FIXED: Return a standard error for duplicate email
		return nil, errors.New("email already registered")
	}

	// 2. Hash password
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}
	hashStr := string(hash)

	// 3. Create User Entity
	user := &entity.User{
		Id:            uuid.New(),
		Email:         req.Email,
		FullName:      req.FullName,
		PasswordHash:  &hashStr,
		Role:          entity.UserRoleUser,
		Status:        entity.UserStatusPending,
		EmailVerified: false,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	// 4. Save to DB
	err = s.userRepo.Create(ctx, user)
	if err != nil {
		return nil, err
	}

	// 5. --- OTP Generation Logic ---
	otpCode, err := generateOTP()
	if err != nil {
		return nil, err
	}

	verificationToken := &entity.EmailVerificationToken{
		Id:        uuid.New(),
		UserId:    user.Id,
		Token:     otpCode,
		ExpiresAt: time.Now().Add(15 * time.Minute), // OTP valid for 15 mins
		CreatedAt: time.Now(),
	}

	err = s.userRepo.CreateVerificationToken(ctx, verificationToken)
	if err != nil {
		return nil, err
	}

	// TODO: Replace this Println with actual Email Service (e.g., SMTP, SendGrid)
	fmt.Printf(">>> [MOCK EMAIL] OTP for %s is: %s <<<\n", user.Email, otpCode)

	return &dto.RegisterResponse{Id: user.Id, Email: user.Email}, nil
}

func (s *authService) VerifyEmail(ctx context.Context, req *dto.VerifyEmailRequest) error {
	user, err := s.userRepo.GetByEmail(ctx, req.Email)
	if err != nil || user == nil {
		return errors.New("user not found")
	}

	if user.Status == entity.UserStatusActive {
		return nil // Already verified, idempotent success
	}

	tokenEntity, err := s.userRepo.GetVerificationToken(ctx, user.Id, req.Token)
	if err != nil {
		return errors.New("invalid otp code")
	}

	if time.Now().After(tokenEntity.ExpiresAt) {
		return errors.New("otp code expired")
	}

	// Activate User
	err = s.userRepo.ActivateUser(ctx, user.Id)
	if err != nil {
		return err
	}

	// Clean up used token
	_ = s.userRepo.DeleteVerificationToken(ctx, tokenEntity.Id)

	return nil
}

func (s *authService) Login(ctx context.Context, req *dto.LoginRequest) (*dto.LoginResponse, error) {
	// 1. Check if user exists
	user, err := s.userRepo.GetByEmail(ctx, req.Email)
	if err != nil || user == nil {
		// Robust handling: return generic error to prevent enumeration
		return nil, errors.New("invalid credentials")
	}

	// 2. Check if user has a password (might be OAuth only)
	if user.PasswordHash == nil {
		return nil, errors.New("user registered via OAuth")
	}

	// Optional: Block login if email is not verified?
	// if user.Status == entity.UserStatusPending {
	// 	return nil, errors.New("please verify your email first")
	// }

	// 3. Compare passwords
	err = bcrypt.CompareHashAndPassword([]byte(*user.PasswordHash), []byte(req.Password))
	if err != nil {
		return nil, errors.New("invalid credentials")
	}

	// 4. Check if user is active
	if user.Status == entity.UserStatusBlocked {
		return nil, errors.New("user account is blocked")
	}

	// 5. Generate JWT
	claims := jwt.MapClaims{
		"user_id": user.Id.String(),
		"role":    user.Role,
		"exp":     time.Now().Add(time.Hour * 24).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		secret = "default_secret" // Fallback for dev
	}
	signedToken, err := token.SignedString([]byte(secret))
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

func (s *authService) ForgotPassword(ctx context.Context, req *dto.ForgotPasswordRequest) error {
	user, err := s.userRepo.GetByEmail(ctx, req.Email)
	if err != nil || user == nil {
		return nil // Fail silently for security
	}

	token := uuid.New().String()
	resetToken := &entity.PasswordResetToken{
		Id:        uuid.New(),
		UserId:    user.Id,
		Token:     token,
		ExpiresAt: time.Now().Add(1 * time.Hour),
		CreatedAt: time.Now(),
		Used:      false,
	}

	err = s.userRepo.CreatePasswordResetToken(ctx, resetToken)
	if err != nil {
		return err
	}

	return nil
}

func (s *authService) ResetPassword(ctx context.Context, req *dto.ResetPasswordRequest) error {
	tokenEntity, err := s.userRepo.GetPasswordResetToken(ctx, req.Token)
	if err != nil {
		return errors.New("invalid or expired token")
	}

	if tokenEntity.Used || time.Now().After(tokenEntity.ExpiresAt) {
		return errors.New("token expired or already used")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	err = s.userRepo.UpdatePassword(ctx, tokenEntity.UserId, string(hash))
	if err != nil {
		return err
	}

	return s.userRepo.MarkTokenUsed(ctx, tokenEntity.Id)
}