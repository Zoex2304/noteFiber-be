// FILE: internal/service/user_service.go
package service

import (
	"ai-notetaking-be/internal/dto"
	"ai-notetaking-be/internal/repository"
	"context"

	"github.com/google/uuid"
)

type IUserService interface {
	GetProfile(ctx context.Context, userId uuid.UUID) (*dto.UserProfileResponse, error)
	UpdateProfile(ctx context.Context, userId uuid.UUID, req *dto.UpdateProfileRequest) error
	DeleteAccount(ctx context.Context, userId uuid.UUID) error
}

type userService struct {
	userRepo repository.IUserRepository
}

func NewUserService(userRepo repository.IUserRepository) IUserService {
	return &userService{userRepo: userRepo}
}

func (s *userService) GetProfile(ctx context.Context, userId uuid.UUID) (*dto.UserProfileResponse, error) {
	user, err := s.userRepo.GetById(ctx, userId)
	if err != nil {
		return nil, err
	}

	return &dto.UserProfileResponse{
		Id:           user.Id,
		Email:        user.Email,
		FullName:     user.FullName,
		Role:         string(user.Role),
		Status:       string(user.Status),
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