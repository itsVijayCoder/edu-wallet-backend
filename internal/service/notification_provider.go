package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/itsVijayCoder/edu-wallet-backend/pkg/email"
)

type NotificationMessage struct {
	Channel   string
	Recipient string
	Subject   string
	Body      string
	Metadata  map[string]any
}

type NotificationResult struct {
	Provider  string
	MessageID string
	Status    string
	Response  map[string]any
}

type NotificationProvider interface {
	Send(ctx context.Context, msg NotificationMessage) (*NotificationResult, error)
}

type notificationProvider struct {
	emailClient *email.Client
}

func NewNotificationProvider(emailClient *email.Client) NotificationProvider {
	return &notificationProvider{emailClient: emailClient}
}

func (p *notificationProvider) Send(ctx context.Context, msg NotificationMessage) (*NotificationResult, error) {
	channel := strings.TrimSpace(msg.Channel)
	switch channel {
	case "email":
		if p.emailClient == nil {
			return &NotificationResult{
				Provider: "resend",
				Status:   "skipped",
				Response: map[string]any{"reason": "email provider is not configured"},
			}, nil
		}
		if err := p.emailClient.Send(ctx, msg.Recipient, msg.Subject, msg.Body); err != nil {
			return nil, err
		}
		return &NotificationResult{
			Provider: "resend",
			Status:   "sent",
			Response: map[string]any{"channel": "email"},
		}, nil
	case "sms", "whatsapp", "in_app":
		return &NotificationResult{
			Provider: "noop",
			Status:   "skipped",
			Response: map[string]any{"reason": fmt.Sprintf("%s provider is not configured", channel)},
		}, nil
	default:
		return nil, fmt.Errorf("unsupported notification channel %q", channel)
	}
}
