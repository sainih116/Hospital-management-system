package notifier

import (
	"context"

	"go.uber.org/zap"
)

// LogNotifier delivers notifications by writing them to the structured
// application log. It is a complete, working Notifier implementation — the
// default delivery channel until an SMTP/SMS-backed Notifier is introduced
// (see backend/.env.example's SMTP_* variables), at which point it is
// swapped in main.go without any caller changes.
type LogNotifier struct {
	log *zap.Logger
}

// NewLogNotifier builds a LogNotifier.
func NewLogNotifier(log *zap.Logger) *LogNotifier {
	return &LogNotifier{log: log}
}

// SendPasswordResetEmail implements Notifier.
func (n *LogNotifier) SendPasswordResetEmail(_ context.Context, toEmail, toName, resetLink string) error {
	n.log.Info("password reset requested",
		zap.String("toEmail", toEmail),
		zap.String("toName", toName),
		zap.String("resetLink", resetLink),
	)
	return nil
}
