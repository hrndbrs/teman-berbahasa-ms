package user

import (
	"context"
	"fmt"

	"github.com/resend/resend-go/v2"
)

type resendUserEmailSender struct {
	client      *resend.Client
	from        string
	frontendURL string
}

func NewEmailSender(apiKey, from, frontendURL string) *resendUserEmailSender {
	return &resendUserEmailSender{
		client:      resend.NewClient(apiKey),
		from:        from,
		frontendURL: frontendURL,
	}
}

func (s *resendUserEmailSender) SendInvite(ctx context.Context, toEmail, firstName, rawToken string) error {
	resetURL := fmt.Sprintf("%s/reset-password?token=%s", s.frontendURL, rawToken)
	_, err := s.client.Emails.Send(&resend.SendEmailRequest{
		From:    s.from,
		To:      []string{toEmail},
		Subject: "Welcome — Set Your Password — Teman Berbahasa",
		Html: fmt.Sprintf(`
<p>Hi %s,</p>
<p>Your Teman Berbahasa account has been created. Click the link below to set your password and get started:</p>
<p><a href="%s">Set Your Password</a></p>
<p>This link expires in <strong>1 hour</strong>.</p>
`, firstName, resetURL),
	})
	return err
}
