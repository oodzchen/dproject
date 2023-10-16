package model

import "time"

const VerficationCodeLifeTime = 5 * time.Minute

type VerificationCode struct {
	Email     string
	Code      string
	CreatedAt *time.Time
	IsUsed    bool
}
