// FILE: internal/service/user_service.go
package service

import (
	"ai-notetaking-be/internal/dto"
	"ai-notetaking-be/internal/repository"
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
)

type IUserService interface {
	GetProfile(ctx context.Context, userId uuid.UUID) (*dto.UserProfileResponse, error)
	UpdateProfile(ctx context.Context, userId uuid.UUID, req *dto.UpdateProfileRequest) error
	DeleteAccount(ctx context.Context, userId uuid.UUID) error
	UploadAvatar(ctx context.Context, userId uuid.UUID, file *multipart.FileHeader) (string, error) // New Method
}

type userService struct {
	userRepo repository.IUserRepository
}

func NewUserService(userRepo repository.IUserRepository) IUserService {
	return &userService{userRepo: userRepo}
}

func (s *userService) GetProfile(ctx context.Context, userId uuid.UUID) (*dto.UserProfileResponse, error) {
	// ✅ CHANGED: Use GetByIdWithAvatar instead of GetById
	user, err := s.userRepo.GetByIdWithAvatar(ctx, userId)
	if err != nil {
		return nil, err
	}

	avatarURL := ""
	if user.AvatarURL != nil {
		avatarURL = *user.AvatarURL
	}

	return &dto.UserProfileResponse{
		Id:           user.Id,
		Email:        user.Email,
		FullName:     user.FullName,
		Role:         string(user.Role),
		Status:       string(user.Status),
		AvatarURL:    avatarURL,    // ✅ NEW: Include avatar URL
		AiDailyUsage: user.AiDailyUsage,
		CreatedAt:    user.CreatedAt,
	}, nil
}

func (s *userService) UpdateProfile(ctx context.Context, userId uuid.UUID, req *dto.UpdateProfileRequest) error {
	user, err := s.userRepo.GetById(ctx, userId)
	if err != nil {
		return err
	}

	user.FullName = req.FullName
	return s.userRepo.Update(ctx, user)
}

func (s *userService) DeleteAccount(ctx context.Context, userId uuid.UUID) error {
	return s.userRepo.Delete(ctx, userId)
}

func (s *userService) UploadAvatar(ctx context.Context, userId uuid.UUID, file *multipart.FileHeader) (string, error) {
	// 1. Validate File Size (e.g., Max 2MB)
	if file.Size > 2*1024*1024 {
		return "", fmt.Errorf("file too large (max 2MB)")
	}

	// 2. Open File
	src, err := file.Open()
	if err != nil {
		return "", err
	}
	defer src.Close()

	// 3. Create Upload Directory
	uploadDir := "./uploads/avatars"
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		return "", err
	}

	// 4. Generate Unique Filename
	ext := filepath.Ext(file.Filename)
	filename := fmt.Sprintf("%s_%d%s", userId.String(), time.Now().Unix(), ext)
	dstPath := filepath.Join(uploadDir, filename)

	// 5. Save File
	dst, err := os.Create(dstPath)
	if err != nil {
		return "", err
	}
	defer dst.Close()

	if _, err = io.Copy(dst, src); err != nil {
		return "", err
	}

	// 6. Generate Public URL (assuming localhost:3000/uploads/...)
	// In production, this base URL should come from ENV
	baseURL := os.Getenv("APP_BASE_URL")
	if baseURL == "" {
		baseURL = "http://localhost:3000"
	}
	publicURL := fmt.Sprintf("%s/uploads/avatars/%s", baseURL, filename)

	// 7. Update User Profile in DB
	err = s.userRepo.UpdateAvatar(ctx, userId, publicURL)
	if err != nil {
		return "", err
	}

	return publicURL, nil
}