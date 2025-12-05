// FILE: internal/service/payment_service.go
package service

import (
	"ai-notetaking-be/internal/dto"
	"ai-notetaking-be/internal/entity"
	"ai-notetaking-be/internal/repository"
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/midtrans/midtrans-go"
	"github.com/midtrans/midtrans-go/snap"
)

type IPaymentService interface {
	GetPlans(ctx context.Context) ([]*dto.PlanResponse, error)
	CreateSubscription(ctx context.Context, userId uuid.UUID, req *dto.CheckoutRequest) (*dto.CheckoutResponse, error)
	HandleNotification(ctx context.Context, req *dto.MidtransWebhookRequest) error
	
	// FIXED: Added missing interface methods
	GetSubscriptionStatus(ctx context.Context, userId uuid.UUID) (*dto.SubscriptionStatusResponse, error)
	CancelSubscription(ctx context.Context, userId uuid.UUID) error
}

type paymentService struct {
	subRepo  repository.ISubscriptionRepository
	userRepo repository.IUserRepository 
}

func NewPaymentService(subRepo repository.ISubscriptionRepository, userRepo repository.IUserRepository) IPaymentService {
	return &paymentService{
		subRepo:  subRepo,
		userRepo: userRepo,
	}
}

func (s *paymentService) GetPlans(ctx context.Context) ([]*dto.PlanResponse, error) {
	plans, err := s.subRepo.GetAllPlans(ctx)
	if err != nil {
		return nil, err
	}

	var res []*dto.PlanResponse
	for _, p := range plans {
		features := []string{"Basic Note Taking"}
		if p.SemanticSearchEnabled {
			features = append(features, "Semantic Search")
		}
		if p.AiChatEnabled {
			features = append(features, "AI Chat Assistant")
		}

		res = append(res, &dto.PlanResponse{
			Id:          p.Id,
			Name:        p.Name,
			Slug:        p.Slug,
			Price:       p.Price,
			Description: p.Description,
			Features:    features,
		})
	}
	return res, nil
}

func (s *paymentService) CreateSubscription(ctx context.Context, userId uuid.UUID, req *dto.CheckoutRequest) (*dto.CheckoutResponse, error) {
	plan, err := s.subRepo.GetPlanById(ctx, req.PlanId)
	if err != nil {
		return nil, err
	}

	user, err := s.userRepo.GetById(ctx, userId)
	if err != nil {
		return nil, errors.New("user not found")
	}

	subId := uuid.New()
	sub := &entity.UserSubscription{
		Id:                 subId,
		UserId:             userId,
		PlanId:             plan.Id,
		Status:             entity.SubscriptionStatusInactive,
		PaymentStatus:      entity.PaymentStatusPending,
		CreatedAt:          time.Now(),
		UpdatedAt:          time.Now(),
		CurrentPeriodStart: time.Now(),
		CurrentPeriodEnd:   time.Now().AddDate(0, 1, 0), 
	}

	if plan.BillingPeriod == entity.BillingPeriodYearly {
		sub.CurrentPeriodEnd = time.Now().AddDate(1, 0, 0)
	}

	err = s.subRepo.CreateSubscription(ctx, sub)
	if err != nil {
		return nil, err
	}

	var sClient snap.Client
	serverKey := os.Getenv("MIDTRANS_SERVER_KEY")
	env := midtrans.Sandbox
	if os.Getenv("MIDTRANS_IS_PRODUCTION") == "true" {
		env = midtrans.Production
	}
	sClient.New(serverKey, env)

	snapReq := &snap.Request{
		TransactionDetails: midtrans.TransactionDetails{
			OrderID:  subId.String(),
			GrossAmt: int64(plan.Price),
		},
		CreditCard: &snap.CreditCardDetails{
			Secure: true,
		},
		CustomerDetail: &midtrans.CustomerDetails{
			FName: user.FullName,
			Email: user.Email,
		},
		Items: &[]midtrans.ItemDetails{
			{
				ID:    plan.Id.String(),
				Price: int64(plan.Price),
				Qty:   1,
				Name:  plan.Name,
			},
		},
		EnabledPayments: snap.AllSnapPaymentType,
	}

	snapResp, midErr := sClient.CreateTransaction(snapReq)
	if midErr != nil {
		return nil, fmt.Errorf("midtrans error: %v", midErr.GetMessage())
	}

	return &dto.CheckoutResponse{
		SubscriptionId:  subId,
		SnapToken:       snapResp.Token,
		SnapRedirectUrl: snapResp.RedirectURL,
	}, nil
}

func (s *paymentService) HandleNotification(ctx context.Context, req *dto.MidtransWebhookRequest) error {
	fmt.Printf("\n[WEBHOOK] Received notification for Order ID: %s | Status: %s\n", req.OrderId, req.TransactionStatus)

	subId, err := uuid.Parse(req.OrderId)
	if err != nil {
		return fmt.Errorf("invalid order id format")
	}

	sub, err := s.subRepo.GetSubscriptionById(ctx, subId)
	if err != nil {
		fmt.Printf("[WEBHOOK] Subscription not found: %s\n", err.Error())
		return err
	}

	var newStatus entity.SubscriptionStatus
	var newPaymentStatus entity.PaymentStatus

	switch req.TransactionStatus {
	case "capture", "settlement":
		newStatus = entity.SubscriptionStatusActive
		newPaymentStatus = entity.PaymentStatusPaid
		fmt.Println("[WEBHOOK] Payment SUCCESS. Activating subscription.")
	case "deny", "cancel", "expire":
		newStatus = entity.SubscriptionStatusInactive
		newPaymentStatus = entity.PaymentStatusFailed
		fmt.Println("[WEBHOOK] Payment FAILED. Canceling subscription.")
	case "pending":
		fmt.Println("[WEBHOOK] Payment PENDING.")
		return nil
	default:
		fmt.Printf("[WEBHOOK] Unknown status: %s\n", req.TransactionStatus)
		return nil
	}

	if sub.Status != newStatus || sub.PaymentStatus != newPaymentStatus {
		return s.subRepo.UpdateSubscriptionStatus(ctx, subId, newStatus, newPaymentStatus)
	}

	return nil
}

func (s *paymentService) GetSubscriptionStatus(ctx context.Context, userId uuid.UUID) (*dto.SubscriptionStatusResponse, error) {
	sub, plan, err := s.subRepo.GetActiveByUserId(ctx, userId)
	if err != nil {
		return nil, err
	}
	
	if sub == nil {
		return &dto.SubscriptionStatusResponse{
			PlanName: "Free Plan",
			Status:   "inactive",
			IsActive: false,
		}, nil
	}

	return &dto.SubscriptionStatusResponse{
		PlanName:           plan.Name,
		Status:             string(sub.Status),
		CurrentPeriodEnd:   sub.CurrentPeriodEnd,
		AiDailyCreditLimit: plan.AiDailyCreditLimit,
		IsActive:           true,
	}, nil
}

func (s *paymentService) CancelSubscription(ctx context.Context, userId uuid.UUID) error {
	sub, _, err := s.subRepo.GetActiveByUserId(ctx, userId)
	if err != nil {
		return err
	}
	if sub == nil {
		return errors.New("no active subscription found")
	}

	// In a real app, you might want to call Midtrans to cancel if it's recurring on their side too
	return s.subRepo.CancelSubscription(ctx, sub.Id)
}