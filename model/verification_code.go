package model

import "time"

type VerificationCode struct {
	Email     string
	Code      string
	CreatedAt *time.Time
	IsUsed    bool
}
