// FILE: internal/entity/subscription_entity.go
package entity

import (
	"time"

	"github.com/google/uuid"
)

type SubscriptionStatus string
type PaymentStatus string
type BillingPeriod string

const (
	SubscriptionStatusActive   SubscriptionStatus = "active"
	SubscriptionStatusInactive SubscriptionStatus = "inactive"
	SubscriptionStatusCanceled SubscriptionStatus = "canceled"

	PaymentStatusPending PaymentStatus = "pending"
	PaymentStatusPaid    PaymentStatus = "success" // CHANGED: Must match DB Enum 'success'
	PaymentStatusFailed  PaymentStatus = "failed"

	BillingPeriodMonthly BillingPeriod = "monthly"
	BillingPeriodYearly  BillingPeriod = "yearly"
)

type SubscriptionPlan struct {
	Id                    uuid.UUID
	Name                  string
	Slug                  string
	Description           string
	Price                 float64
	BillingPeriod         BillingPeriod
	MaxNotes              *int
	SemanticSearchEnabled bool
	AiChatEnabled         bool
	AiDailyCreditLimit    int
}

type UserSubscription struct {
	Id                    uuid.UUID
	UserId                uuid.UUID
	PlanId                uuid.UUID
	BillingAddressId      *uuid.UUID // Added field
	Status                SubscriptionStatus
	CurrentPeriodStart    time.Time
	CurrentPeriodEnd      time.Time
	PaymentStatus         PaymentStatus
	MidtransTransactionId *string
	CreatedAt             time.Time
	UpdatedAt             time.Time
}