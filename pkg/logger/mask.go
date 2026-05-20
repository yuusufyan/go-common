package logger

import (
	"strings"

	"github.com/sirupsen/logrus"
)

// MaskHook is a logrus hook that masks sensitive data in log fields
type MaskHook struct {
	SensitiveKeys []string
}

// Levels returns the levels at which this hook should fire
func (h *MaskHook) Levels() []logrus.Level {
	return logrus.AllLevels
}

// Fire is called for each log entry
func (h *MaskHook) Fire(entry *logrus.Entry) error {
	for _, key := range h.SensitiveKeys {
		if _, ok := entry.Data[key]; ok {
			entry.Data[key] = "********"
		}
		// Also check case-insensitive or nested if needed, 
		// but simple key match is a good start.
		for k, v := range entry.Data {
			if strings.EqualFold(k, key) {
				entry.Data[k] = "********"
			}
			// If value is a map, we could recurse, but for simplicity:
			_ = v
		}
	}
	return nil
}

// NewMaskHook creates a new hook with default sensitive keys
func NewMaskHook() *MaskHook {
	return &MaskHook{
		SensitiveKeys: []string{"password", "token", "secret", "refresh_token", "access_token", "authorization"},
	}
}
