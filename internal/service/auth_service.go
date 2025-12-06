// FILE: internal/service/auth_service.go
package service

import (
	"ai-notetaking-be/internal/dto"
	"ai-notetaking-be/internal/entity"
	"ai-notetaking-be/internal/pkg/mailer"
	"ai-notetaking-be/internal/repository"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
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
	Login(ctx context.Context, req *dto.LoginRequest, ipAddress, userAgent string) (*dto.LoginResponse, error) // Updated signature
	ForgotPassword(ctx context.Context, req *dto.ForgotPasswordRequest) error
	ResetPassword(ctx context.Context, req *dto.ResetPasswordRequest) error
	VerifyEmail(ctx context.Context, req *dto.VerifyEmailRequest) error
	Logout(ctx context.Context, refreshToken string) error // ✅ NEW METHOD from code pembaharuan
}

type authService struct {
	userRepo     repository.IUserRepository
	emailService mailer.IEmailService
}

func NewAuthService(userRepo repository.IUserRepository, emailService mailer.IEmailService) IAuthService {
	return &authService{
		userRepo:     userRepo,
		emailService: emailService,
	}
}

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

	// 5. Generate and save OTP
	otpCode, err := generateOTP()
	if err != nil {
		return nil, err
	}

	verificationToken := &entity.EmailVerificationToken{
		Id:        uuid.New(),
		UserId:    user.Id,
		Token:     otpCode,
		ExpiresAt: time.Now().Add(15 * time.Minute),
		CreatedAt: time.Now(),
	}

	err = s.userRepo.CreateVerificationToken(ctx, verificationToken)
	if err != nil {
		return nil, err
	}

	// Log to console for dev convenience
	fmt.Printf(">>> [DEBUG OTP] OTP for %s is: %s <<<\n", user.Email, otpCode)

	// SEND REAL EMAIL
	// We run this in a goroutine so the API response isn't delayed by SMTP latency
	go func() {
		emailErr := s.emailService.SendOTP(user.Email, otpCode)
		if emailErr != nil {
			fmt.Printf("Error sending registration email: %v\n", emailErr)
		}
	}()

	return &dto.RegisterResponse{Id: user.Id, Email: user.Email}, nil
}

func (s *authService) VerifyEmail(ctx context.Context, req *dto.VerifyEmailRequest) error {
	user, err := s.userRepo.GetByEmail(ctx, req.Email)
	if err != nil || user == nil {
		return errors.New("user not found")
	}

	if user.Status == entity.UserStatusActive {
		return nil
	}

	tokenEntity, err := s.userRepo.GetVerificationToken(ctx, user.Id, req.Token)
	if err != nil {
		return errors.New("invalid otp code")
	}

	if time.Now().After(tokenEntity.ExpiresAt) {
		return errors.New("otp code expired")
	}

	err = s.userRepo.ActivateUser(ctx, user.Id)
	if err != nil {
		return err
	}

	_ = s.userRepo.DeleteVerificationToken(ctx, tokenEntity.Id)

	return nil
}

func (s *authService) Login(ctx context.Context, req *dto.LoginRequest, ipAddress, userAgent string) (*dto.LoginResponse, error) {
	// 1. Check if user exists
	user, err := s.userRepo.GetByEmail(ctx, req.Email)
	if err != nil || user == nil {
		return nil, errors.New("invalid credentials")
	}

	// 2. Check if user has a password (might be OAuth only)
	if user.PasswordHash == nil {
		return nil, errors.New("user registered via OAuth")
	}

	// 3. SECURITY CHECK: Check if email is verified
	// If status is pending or email_verified is false, reject login.
	if user.Status == entity.UserStatusPending || !user.EmailVerified {
		return nil, errors.New("email not verified. please check your inbox for the otp code")
	}

	// 4. Compare passwords
	err = bcrypt.CompareHashAndPassword([]byte(*user.PasswordHash), []byte(req.Password))
	if err != nil {
		return nil, errors.New("invalid credentials")
	}

	// 5. Check if user is blocked/suspended
	if user.Status == entity.UserStatusBlocked {
		return nil, errors.New("user account is blocked")
	}

	// 6. Generate JWT
	// Default Access Token Expiry (24 hours)
	// You might want to shorten this to e.g., 15 mins if you rely on refresh tokens
	accessTokenExpiry := time.Hour * 24

	claims := jwt.MapClaims{
		"user_id": user.Id.String(),
		"role":    user.Role,
		"exp":     time.Now().Add(accessTokenExpiry).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		secret = "default_secret"
	}
	signedToken, err := token.SignedString([]byte(secret))
	if err != nil {
		return nil, err
	}

	var rawRefreshToken string

	// Hanya buat Refresh Token jika "Remember Me" dicentang - FROM PEMBAHARUAN
	if req.RememberMe {
		rawRefreshToken = uuid.New().String()
		
		// Hash token
		hasher := sha256.New()
		hasher.Write([]byte(rawRefreshToken))
		tokenHash := hex.EncodeToString(hasher.Sum(nil))

		// Set Expiry: 30 Hari
		refreshTokenEntity := &entity.UserRefreshToken{
			Id:        uuid.New(),
			UserId:    user.Id,
			TokenHash: tokenHash,
			ExpiresAt: time.Now().Add(time.Hour * 24 * 30), 
			Revoked:   false,
			CreatedAt: time.Now(),
			IpAddress: ipAddress,
			UserAgent: userAgent,
		}

		err = s.userRepo.CreateRefreshToken(ctx, refreshTokenEntity)
		if err != nil {
			return nil, fmt.Errorf("failed to create session: %v", err)
		}
	}

	return &dto.LoginResponse{
		AccessToken:  signedToken,
		RefreshToken: rawRefreshToken, // Empty string if not requested - FROM PEMBAHARUAN
		User: dto.UserDTO{
			Id:       user.Id,
			Email:    user.Email,
			FullName: user.FullName,
			Role:     string(user.Role),
		},
	}, nil
}

// ✅ NEW LOGOUT IMPLEMENTATION from code pembaharuan
func (s *authService) Logout(ctx context.Context, refreshToken string) error {
	// If no token is provided, just return (stateless logout)
	if refreshToken == "" {
		return nil
	}

	// Hash the token to find it in DB
	hasher := sha256.New()
	hasher.Write([]byte(refreshToken))
	tokenHash := hex.EncodeToString(hasher.Sum(nil))

	// Mark as revoked
	return s.userRepo.RevokeRefreshToken(ctx, tokenHash)
}

func (s *authService) ForgotPassword(ctx context.Context, req *dto.ForgotPasswordRequest) error {
	user, err := s.userRepo.GetByEmail(ctx, req.Email)
	if err != nil || user == nil {
		return nil
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

	go func() {
		emailErr := s.emailService.SendResetToken(user.Email, token)
		if emailErr != nil {
			fmt.Printf("Error sending reset password email: %v\n", emailErr)
		}
	}()

	return nil
}

func (s *authService) ResetPassword(ctx context.Context, req *dto.ResetPasswordRequest) error {
	tokenEntity, err := s.userRepo.GetPasswordResetToken(ctx, req.Token)
	if err != nil {
		return errors.New("invalid or expired token")
	}

	// 1. Check if explicitly marked as used
	if tokenEntity.Used {
		return errors.New("this password reset link has already been used")
	}

	// 2. Check expiry
	if time.Now().After(tokenEntity.ExpiresAt) {
		return errors.New("this password reset link has expired")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	err = s.userRepo.UpdatePassword(ctx, tokenEntity.UserId, string(hash))
	if err != nil {
		return err
	}

	// 3. Mark as used so it cannot be used again
	return s.userRepo.MarkTokenUsed(ctx, tokenEntity.Id)
}