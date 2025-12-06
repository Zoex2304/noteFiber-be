// FILE: internal/service/admin_service.go
package service

import (
	"ai-notetaking-be/internal/dto"
	"ai-notetaking-be/internal/entity"
	"ai-notetaking-be/internal/pkg/logger" // This should now work
	"ai-notetaking-be/internal/repository"
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type IAdminService interface {
	GetDashboardStats(ctx context.Context) (*dto.AdminDashboardStats, error)
	GetUserGrowth(ctx context.Context) ([]*dto.UserGrowthStats, error)
	
	// User Management
	GetAllUsers(ctx context.Context, page, limit int, search string) ([]*dto.UserListResponse, error)
	GetUserDetail(ctx context.Context, userId uuid.UUID) (*dto.UserProfileResponse, error)
	UpdateUserStatus(ctx context.Context, userId uuid.UUID, status string) error
	
	// Transaction Management
	GetTransactions(ctx context.Context, page, limit int, status string) ([]*dto.TransactionListResponse, error)
	
	// Logs
	GetSystemLogs(ctx context.Context, page, limit int, level string) ([]*dto.LogListResponse, error)
	GetLogDetail(ctx context.Context, logId uuid.UUID) (*dto.LogDetailResponse, error)
}

type adminService struct {
	userRepo repository.IUserRepository
	subRepo  repository.ISubscriptionRepository
	logger   logger.ILogger 
}

func NewAdminService(
	userRepo repository.IUserRepository, 
	subRepo repository.ISubscriptionRepository,
	logger logger.ILogger,
) IAdminService {
	return &adminService{
		userRepo: userRepo,
		subRepo:  subRepo,
		logger:   logger,
	}
}

func (s *adminService) GetDashboardStats(ctx context.Context) (*dto.AdminDashboardStats, error) {
	totalUsers, err := s.userRepo.Count(ctx)
	if err != nil { return nil, err }

	activeUsers, err := s.userRepo.CountByStatus(ctx, entity.UserStatusActive)
	if err != nil { return nil, err }

	totalRevenue, err := s.subRepo.GetTotalRevenue(ctx)
	if err != nil { return nil, err }

	activeSubs, err := s.subRepo.CountActiveSubscribers(ctx)
	if err != nil { return nil, err }

	return &dto.AdminDashboardStats{
		TotalUsers:        totalUsers,
		ActiveUsers:       activeUsers,
		TotalRevenue:      totalRevenue,
		ActiveSubscribers: activeSubs,
	}, nil
}

func (s *adminService) GetUserGrowth(ctx context.Context) ([]*dto.UserGrowthStats, error) {
	stats, err := s.userRepo.GetUserGrowth(ctx)
	if err != nil {
		return nil, err
	}
	var res []*dto.UserGrowthStats
	for _, st := range stats {
		res = append(res, &dto.UserGrowthStats{
			Date:  st["date"].(string),
			Count: int(st["count"].(int64)),
		})
	}
	return res, nil
}

func (s *adminService) GetAllUsers(ctx context.Context, page, limit int, search string) ([]*dto.UserListResponse, error) {
	if page < 1 { page = 1 }
	if limit < 1 { limit = 10 }
	offset := (page - 1) * limit

	var users []*entity.User
	var err error

	if search != "" {
		users, err = s.userRepo.SearchUsers(ctx, search, limit, offset)
	} else {
		users, err = s.userRepo.GetAll(ctx, limit, offset)
	}
	
	if err != nil {
		return nil, err
	}

	var res []*dto.UserListResponse
	for _, u := range users {
		res = append(res, &dto.UserListResponse{
			Id:        u.Id,
			Email:     u.Email,
			FullName:  u.FullName,
			Role:      string(u.Role),
			Status:    string(u.Status),
			CreatedAt: u.CreatedAt,
		})
	}
	return res, nil
}

func (s *adminService) GetUserDetail(ctx context.Context, userId uuid.UUID) (*dto.UserProfileResponse, error) {
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

func (s *adminService) UpdateUserStatus(ctx context.Context, userId uuid.UUID, status string) error {
	s.logger.Info("ADMIN", "Updated user status", map[string]interface{}{
		"userId": userId.String(),
		"status": status,
		"admin":  "system", 
	})
	return s.userRepo.UpdateStatus(ctx, userId, entity.UserStatus(status))
}

func (s *adminService) GetTransactions(ctx context.Context, page, limit int, status string) ([]*dto.TransactionListResponse, error) {
	if page < 1 { page = 1 }
	if limit < 1 { limit = 10 }
	offset := (page - 1) * limit

	txs, err := s.subRepo.GetTransactions(ctx, status, limit, offset)
	if err != nil {
		return nil, err
	}

	var res []*dto.TransactionListResponse
	for _, t := range txs {
		res = append(res, &dto.TransactionListResponse{
			Id:              t.Id,
			UserId:          t.UserId,
			UserEmail:       t.UserEmail,
			PlanName:        t.PlanName,
			Amount:          t.Amount,
			Status:          t.Status,
			PaymentStatus:   t.PaymentStatus,
			TransactionDate: t.CreatedAt,
			MidtransOrderId: t.MidtransOrderId,
		})
	}
	return res, nil
}

func (s *adminService) GetSystemLogs(ctx context.Context, page, limit int, level string) ([]*dto.LogListResponse, error) {
	logs, err := s.logger.GetLogs(level, limit, (page-1)*limit)
	if err != nil {
		return nil, err
	}

	var res []*dto.LogListResponse
	for _, l := range logs {
		ts, _ := time.Parse(time.RFC3339, l.Timestamp)
		res = append(res, &dto.LogListResponse{
			Id:        uuid.MustParse(l.Id), 
			Level:     l.Level,
			Module:    l.Module,
			Message:   l.Message,
			CreatedAt: ts,
		})
	}
	return res, nil
}

func (s *adminService) GetLogDetail(ctx context.Context, logId uuid.UUID) (*dto.LogDetailResponse, error) {
	l, err := s.logger.GetLogById(logId.String())
	if err != nil {
		return nil, err
	}

	ts, _ := time.Parse(time.RFC3339, l.Timestamp)
	var detailsMap map[string]interface{}
	json.Unmarshal([]byte(l.Details), &detailsMap)

	return &dto.LogDetailResponse{
		LogListResponse: dto.LogListResponse{
			Id:        logId,
			Level:     l.Level,
			Module:    l.Module,
			Message:   l.Message,
			CreatedAt: ts,
		},
		Details: detailsMap,
	}, nil
}