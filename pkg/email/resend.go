package email

import (
	"context"
	"fmt"

	"github.com/resend/resend-go/v2"
)

// Client wraps the Resend email API.
// If the API key is empty, Send is a no-op (graceful degradation).
type Client struct {
	client    *resend.Client
	fromEmail string
	fromName  string
}

func NewClient(apiKey, fromEmail, fromName string) *Client {
	var c *resend.Client
	if apiKey != "" {
		c = resend.NewClient(apiKey)
	}
	return &Client{
		client:    c,
		fromEmail: fromEmail,
		fromName:  fromName,
	}
}

func (c *Client) Send(ctx context.Context, to, subject, html string) error {
	if c.client == nil {
		return nil // no-op when unconfigured
	}

	from := fmt.Sprintf("%s <%s>", c.fromName, c.fromEmail)
	_, err := c.client.Emails.SendWithContext(ctx, &resend.SendEmailRequest{
		From:    from,
		To:      []string{to},
		Subject: subject,
		Html:    html,
	})
	return err
}
