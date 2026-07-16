package service

import (
	"context"
	"fmt"
)

// OTPNotifier delivers a one-time password without coupling authentication to
// a particular SMS vendor. Production wiring uses NotificationProvider; tests
// inject a deterministic notifier.
type OTPNotifier interface {
	SendOTP(ctx context.Context, phone, otp string) error
}

type notificationOTPNotifier struct {
	provider NotificationProvider
}

type noopOTPNotifier struct{}

// NewNoopOTPNotifier is suitable only for tests that inspect the generated
// code from Redis instead of delivering it to an external SMS gateway.
func NewNoopOTPNotifier() OTPNotifier {
	return noopOTPNotifier{}
}

func (noopOTPNotifier) SendOTP(context.Context, string, string) error { return nil }

func NewNotificationOTPNotifier(provider NotificationProvider) OTPNotifier {
	return &notificationOTPNotifier{provider: provider}
}

func (n *notificationOTPNotifier) SendOTP(ctx context.Context, phone, otp string) error {
	if n.provider == nil {
		return fmt.Errorf("SMS OTP provider is not configured")
	}

	result, err := n.provider.Send(ctx, NotificationMessage{
		Channel:   "sms",
		Recipient: phone,
		Body:      fmt.Sprintf("Your EduWallet verification code is %s. It expires in 5 minutes.", otp),
	})
	if err != nil {
		return fmt.Errorf("send SMS OTP: %w", err)
	}
	if result == nil || result.Status != "sent" {
		return fmt.Errorf("SMS OTP delivery was not accepted by the provider")
	}
	return nil
}
