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
	GetOrderSummary(ctx context.Context, planId uuid.UUID) (*dto.OrderSummaryResponse, error)
	CreateSubscription(ctx context.Context, userId uuid.UUID, req *dto.CheckoutRequest) (*dto.CheckoutResponse, error)
	HandleNotification(ctx context.Context, req *dto.MidtransWebhookRequest) error
	GetSubscriptionStatus(ctx context.Context, userId uuid.UUID) (*dto.SubscriptionStatusResponse, error)
	CancelSubscription(ctx context.Context, userId uuid.UUID) error
}

type paymentService struct {
	subRepo     repository.ISubscriptionRepository
	userRepo    repository.IUserRepository
	billingRepo repository.IBillingRepository
}

func NewPaymentService(
	subRepo repository.ISubscriptionRepository, 
	userRepo repository.IUserRepository,
	billingRepo repository.IBillingRepository,
) IPaymentService {
	return &paymentService{
		subRepo:     subRepo,
		userRepo:    userRepo,
		billingRepo: billingRepo,
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

// IMPLEMENTATION OF GetOrderSummary
func (s *paymentService) GetOrderSummary(ctx context.Context, planId uuid.UUID) (*dto.OrderSummaryResponse, error) {
	plan, err := s.subRepo.GetPlanById(ctx, planId)
	if err != nil {
		return nil, err
	}

	// 1. Get Base Price
	subtotal := plan.Price

	// 2. Get Tax Rate from Database
	taxRate := plan.TaxRate

	// 3. Calculate Logic
	tax := subtotal * taxRate
	total := subtotal + tax

	billingPeriod := "month"
	if plan.BillingPeriod == entity.BillingPeriodYearly {
		billingPeriod = "year"
	}

	return &dto.OrderSummaryResponse{
		PlanName:      plan.Name,
		BillingPeriod: billingPeriod,
		PricePerUnit:  fmt.Sprintf("$%.2f/%s", plan.Price, billingPeriod),
		Subtotal:      subtotal,
		Tax:           tax,
		Total:         total,
		Currency:      "USD", 
	}, nil
}

func (s *paymentService) CreateSubscription(ctx context.Context, userId uuid.UUID, req *dto.CheckoutRequest) (*dto.CheckoutResponse, error) {
	// 1. Get Plan and User
	plan, err := s.subRepo.GetPlanById(ctx, req.PlanId)
	if err != nil {
		return nil, err
	}

	// Verify user exists (using blank identifier)
	_, err = s.userRepo.GetById(ctx, userId)
	if err != nil {
		return nil, errors.New("user not found")
	}

	// 2. Create Billing Address
	billingId := uuid.New()
	billingAddr := &entity.BillingAddress{
		Id:           billingId,
		UserId:       userId,
		FirstName:    req.FirstName,
		LastName:     req.LastName,
		Email:        req.Email,
		Phone:        req.Phone,
		AddressLine1: req.AddressLine1,
		AddressLine2: req.AddressLine2,
		City:         req.City,
		State:        req.State,
		PostalCode:   req.PostalCode,
		Country:      req.Country,
		IsDefault:    true,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	if err := s.billingRepo.Create(ctx, billingAddr); err != nil {
		return nil, fmt.Errorf("failed to save billing address: %v", err)
	}

	// 3. Create Subscription Record
	subId := uuid.New()
	sub := &entity.UserSubscription{
		Id:                 subId,
		UserId:             userId,
		PlanId:             plan.Id,
		BillingAddressId:   &billingId,
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

	if err := s.subRepo.CreateSubscription(ctx, sub); err != nil {
		return nil, err
	}

	// 4. Initiate Midtrans Transaction
	var sClient snap.Client
	serverKey := os.Getenv("MIDTRANS_SERVER_KEY")
	env := midtrans.Sandbox
	if os.Getenv("MIDTRANS_IS_PRODUCTION") == "true" {
		env = midtrans.Production
	}
	sClient.New(serverKey, env)

	// Get Frontend URL from ENV for dynamic redirect
	frontendURL := os.Getenv("FRONTEND_URL")
	
	// Create callback URLs using the ENV variable
	finishRedirectURL := fmt.Sprintf("%s/app?payment=success", frontendURL)

	// Calculate Final Amount with Tax from DB
	taxRate := plan.TaxRate 
	finalAmount := int64(plan.Price + (plan.Price * taxRate))

	snapReq := &snap.Request{
		TransactionDetails: midtrans.TransactionDetails{
			OrderID:  subId.String(),
			GrossAmt: finalAmount, // Updated to use Total (incl. Tax)
		},
		CreditCard: &snap.CreditCardDetails{
			Secure: true,
		},
		Callbacks: &snap.Callbacks{
			Finish:      finishRedirectURL,
		},
		CustomerDetail: &midtrans.CustomerDetails{
			FName:    req.FirstName,
			LName:    req.LastName,
			Email:    req.Email,
			Phone:    req.Phone,
			BillAddr: &midtrans.CustomerAddress{
				FName:       req.FirstName,
				LName:       req.LastName,
				Phone:       req.Phone,
				Address:     req.AddressLine1,
				City:        req.City,
				Postcode:    req.PostalCode,
				CountryCode: "IDN",
			},
		},
		Items: &[]midtrans.ItemDetails{
			{
				ID:    plan.Id.String(),
				Price: int64(plan.Price), // Base Price
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
	return s.subRepo.CancelSubscription(ctx, sub.Id)
}