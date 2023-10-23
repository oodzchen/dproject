package service

type Service struct {
	Article         *Article
	User            *User
	Permission      *Permission
	UserLogger      *UserLogger
	Verifier        *Verifier
	Mail            *Mail
	SettingsManager *SettingsManager
}
