package main

import (
	"github.com/oodzchen/dproject/config"
	"golang.org/x/crypto/acme/autocert"
)

func NewCertManager() *autocert.Manager {
	return &autocert.Manager{
		Cache:  autocert.DirCache("tls-cache"),
		Prompt: autocert.AcceptTOS,
		Email:  config.Config.AdminEmail,
		// HostPolicy: autocert.HostWhitelist(config.Config.DomainName),
	}
}
