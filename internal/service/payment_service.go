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
}

type paymentService struct {
	subRepo  repository.ISubscriptionRepository
	userRepo repository.IUserRepository // Added to fetch user details for Snap
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
		// Basic features list based on plan type
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
	// 1. Get Plan Details
	plan, err := s.subRepo.GetPlanById(ctx, req.PlanId)
	if err != nil {
		return nil, err
	}

	// 2. Get User Details (for Midtrans Customer Details)
	user, err := s.userRepo.GetById(ctx, userId)
	if err != nil {
		return nil, errors.New("user not found")
	}

	// 3. Create Pending Subscription Record in DB
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
		CurrentPeriodEnd:   time.Now().AddDate(0, 1, 0), // Default monthly
	}

	if plan.BillingPeriod == entity.BillingPeriodYearly {
		sub.CurrentPeriodEnd = time.Now().AddDate(1, 0, 0)
	}

	err = s.subRepo.CreateSubscription(ctx, sub)
	if err != nil {
		return nil, err
	}

	// 4. Initialize Midtrans Snap Client
	var sClient snap.Client
	serverKey := os.Getenv("MIDTRANS_SERVER_KEY")
	env := midtrans.Sandbox
	if os.Getenv("MIDTRANS_IS_PRODUCTION") == "true" {
		env = midtrans.Production
	}
	sClient.New(serverKey, env)

	// 5. Build Snap Request
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

	// 6. Create Transaction
	// Use midErr to avoid shadowing 'err' (which is type error) and forcing type conversion
	snapResp, midErr := sClient.CreateTransaction(snapReq)
	if midErr != nil {
		return nil, fmt.Errorf("midtrans error: %v", midErr.GetMessage())
	}

	// 7. Save Snap Token to DB (Optional, but good for reference)
	// We could update the subscription record with the transaction ID here if needed

	return &dto.CheckoutResponse{
		SubscriptionId:  subId,
		SnapToken:       snapResp.Token,
		SnapRedirectUrl: snapResp.RedirectURL,
	}, nil
}

func (s *paymentService) HandleNotification(ctx context.Context, req *dto.MidtransWebhookRequest) error {
	// Log webhook receipt
	fmt.Printf("\n[WEBHOOK] Received notification for Order ID: %s | Status: %s\n", req.OrderId, req.TransactionStatus)

	// Midtrans sends OrderID which corresponds to our UserSubscription ID
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

	// Logic based on Midtrans transaction status
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
		return nil // Do nothing for pending
	default:
		fmt.Printf("[WEBHOOK] Unknown status: %s\n", req.TransactionStatus)
		return nil // Unknown status, ignore
	}

	// Update DB only if status changed
	if sub.Status != newStatus || sub.PaymentStatus != newPaymentStatus {
		return s.subRepo.UpdateSubscriptionStatus(ctx, subId, newStatus, newPaymentStatus)
	}

	return nil
}