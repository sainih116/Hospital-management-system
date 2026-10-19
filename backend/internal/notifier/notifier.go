// Package notifier defines the outbound notification channel used by
// modules that need to reach a user outside the API (password resets,
// appointment reminders, lab-result alerts, ...). Swapping the delivery
// mechanism (SMTP, SMS gateway, push) never requires touching callers —
// only the constructor wired in main.go changes.
package notifier

import "context"

// Notifier delivers durable, out-of-band notifications to users.
type Notifier interface {
	// SendPasswordResetEmail delivers a password-reset link to a user.
	SendPasswordResetEmail(ctx context.Context, toEmail, toName, resetLink string) error
}
