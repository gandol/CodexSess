package httpapi

import (
	"errors"
	"strings"
)

func ResolveAccountHeader(raw string) (string, error) {
	v := strings.TrimSpace(raw)
	if v == "" {
		return "", errors.New("X-Codex-Account is required")
	}
	return v, nil
}

func BearerToken(raw string) string {
	parts := strings.SplitN(strings.TrimSpace(raw), " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return ""
	}
	return strings.TrimSpace(parts[1])
}
