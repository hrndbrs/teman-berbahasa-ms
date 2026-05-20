package auth

import (
	"context"
	"fmt"

	"github.com/resend/resend-go/v2"
)

type resendEmailSender struct {
	client      *resend.Client
	from        string
	frontendURL string
}

func NewEmailSender(apiKey, from, frontendURL string) *resendEmailSender {
	return &resendEmailSender{
		client:      resend.NewClient(apiKey),
		from:        from,
		frontendURL: frontendURL,
	}
}

func (s *resendEmailSender) SendPasswordReset(ctx context.Context, toEmail, rawToken string) error {
	resetURL := fmt.Sprintf("%s/reset-password?token=%s", s.frontendURL, rawToken)
	_, err := s.client.Emails.Send(&resend.SendEmailRequest{
		From:    s.from,
		To:      []string{toEmail},
		Subject: "Reset Your Password — Teman Berbahasa",
		Html: fmt.Sprintf(`
<p>You requested a password reset for your Teman Berbahasa account.</p>
<p><a href="%s">Click here to reset your password</a></p>
<p>This link expires in <strong>1 hour</strong>. If you did not request this, you can safely ignore this email.</p>
`, resetURL),
	})
	return err
}
