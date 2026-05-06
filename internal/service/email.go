package service

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/itsVijayCoder/edu-wallet-backend/pkg/email"
)

type emailService struct {
	client      *email.Client
	externalURL string
	log         *slog.Logger
}

func NewEmailService(client *email.Client, externalURL string, log *slog.Logger) EmailService {
	return &emailService{client: client, externalURL: externalURL, log: log}
}

func (s *emailService) SendPasswordReset(ctx context.Context, to, token string) error {
	resetURL := fmt.Sprintf("%s/reset-password?token=%s", s.externalURL, token)
	html := fmt.Sprintf(`
		<h2>Password Reset</h2>
		<p>Click the link below to reset your password:</p>
		<p><a href="%s">Reset Password</a></p>
		<p>This link expires in 1 hour.</p>
		<p>If you didn't request this, you can safely ignore this email.</p>
	`, resetURL)

	if err := s.client.Send(ctx, to, "Password Reset", html); err != nil {
		s.log.Error("failed to send password reset email", "to", to, "error", err)
		return err
	}
	return nil
}

func (s *emailService) SendWelcome(ctx context.Context, to, name string) error {
	html := fmt.Sprintf(`
		<h2>Welcome, %s!</h2>
		<p>Your account has been created successfully.</p>
	`, name)

	if err := s.client.Send(ctx, to, "Welcome", html); err != nil {
		s.log.Error("failed to send welcome email", "to", to, "error", err)
		return err
	}
	return nil
}
