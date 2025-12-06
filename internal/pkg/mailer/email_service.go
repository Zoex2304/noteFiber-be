// FILE: internal/pkg/mailer/email_service.go
package mailer

import (
	"fmt"
	"os"

	"gopkg.in/gomail.v2"
)

type IEmailService interface {
	SendOTP(toEmail, otp string) error
	SendResetToken(toEmail, token string) error
}

type emailService struct {
	dialer      *gomail.Dialer
	senderEmail string
	frontendURL string // Added to construct links
}

func NewEmailService(host string, port int, username, password, senderEmail string) IEmailService {
	d := gomail.NewDialer(host, port, username, password)
	
	// Get Frontend URL from ENV or default to a safe placeholder
	frontendURL := os.Getenv("FRONTEND_URL")
	
	return &emailService{
		dialer:      d,
		senderEmail: senderEmail,
		frontendURL: frontendURL,
	}
}

func (s *emailService) SendOTP(toEmail, otp string) error {
	m := gomail.NewMessage()
	m.SetHeader("From", s.senderEmail)
	m.SetHeader("To", toEmail)
	m.SetHeader("Subject", "Your Verification Code")
	
	body := fmt.Sprintf(`
		<div style="font-family: Arial, sans-serif; padding: 20px; color: #333;">
			<h2>Welcome to NoteFiber!</h2>
			<p>Your verification code is:</p>
			<h1 style="color: #4CAF50; letter-spacing: 5px;">%s</h1>
			<p>This code will expire in 15 minutes.</p>
			<p>If you didn't request this, please ignore this email.</p>
		</div>
	`, otp)
	
	m.SetBody("text/html", body)

	if err := s.dialer.DialAndSend(m); err != nil {
		fmt.Printf("[MAILER ERROR] Failed to send OTP to %s: %v\n", toEmail, err)
		return err
	}
	
	fmt.Printf("[MAILER] OTP sent to %s\n", toEmail)
	return nil
}

func (s *emailService) SendResetToken(toEmail, token string) error {
	m := gomail.NewMessage()
	m.SetHeader("From", s.senderEmail)
	m.SetHeader("To", toEmail)
	m.SetHeader("Subject", "Reset Your Password")

	// Construct the clickable link pointing to the FRONTEND
	resetLink := fmt.Sprintf("%s/reset-password?token=%s", s.frontendURL, token)

	body := fmt.Sprintf(`
		<div style="font-family: Arial, sans-serif; padding: 20px; color: #333;">
			<h2>Password Reset Request</h2>
			<p>You requested to reset your password. Click the button below to proceed:</p>
			<a href="%s" style="background-color: #007BFF; color: white; padding: 10px 20px; text-decoration: none; border-radius: 5px; display: inline-block;">Reset Password</a>
			<p>Or copy this link:</p>
			<p>%s</p>
			<p>This link will expire in 1 hour.</p>
			<p>If you didn't request this, please ignore this email.</p>
		</div>
	`, resetLink, resetLink)

	m.SetBody("text/html", body)

	if err := s.dialer.DialAndSend(m); err != nil {
		fmt.Printf("[MAILER ERROR] Failed to send Reset Token to %s: %v\n", toEmail, err)
		return err
	}

	fmt.Printf("[MAILER] Reset Token sent to %s\n", toEmail)
	return nil
}