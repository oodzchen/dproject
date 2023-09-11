package main

import (
	"errors"

	"github.com/oodzchen/dproject/config"
	"golang.org/x/crypto/acme/autocert"
)

const DefaultACMEDirectory = "https://acme-v02.api.letsencrypt.org/directory"

var ErrCacheMiss = errors.New("acme/autocert: certificate cache miss")

func NewCertManager() *autocert.Manager {
	return &autocert.Manager{
		Cache:      autocert.DirCache("tls-cache"),
		Prompt:     autocert.AcceptTOS,
		Email:      config.Config.AdminEmail,
		HostPolicy: autocert.HostWhitelist(config.Config.DomainName),
	}
}
