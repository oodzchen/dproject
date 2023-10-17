package service

import (
	"fmt"
	"net"
	"strings"

	"github.com/emersion/go-sasl"
	"github.com/emersion/go-smtp"
)

type Mail struct {
	LoginEmail     string
	auth           sasl.Client
	SMTPServer     string
	SMTPServerPort string
}

func NewMail(userEmail, password string, smtpServer, smtpServerPort string) *Mail {
	auth := sasl.NewPlainClient("", userEmail, password)

	return &Mail{userEmail, auth, smtpServer, smtpServerPort}
}

func (m *Mail) SendVerificationCode(email, code string) error {

	addr := net.JoinHostPort(m.SMTPServer, m.SMTPServerPort)
	msgTpl := "To: %s\r\n"
	msgTpl += "From: %s\r\n"
	msgTpl += "Subject: Verification code\r\n"
	msgTpl += "MIME-version: 1.0;\nContent-Type: text/html; charset=\"UTF-8\";\n\n"
	msgTpl += "<html><body><h1>Verification code</h1>"
	msgTpl += "The code is <b>%s</b>\r\n<br>"
	msgTpl += "For security reasons, please do not disclose the verification code.\r\n<br>"
	msgTpl += "<hr>Kholin\r\n<br>"
	msgTpl += "</body></html>"
	msgData := fmt.Sprintf(msgTpl, email, m.LoginEmail, code)

	// fmt.Println("msg data: ", msgData)

	msg := strings.NewReader(msgData)
	to := []string{email}

	// fmt.Println("smtp addr: ", addr)

	err := smtp.SendMail(addr, m.auth, m.LoginEmail, to, msg)
	// err := smtp.SendMail(email, m.auth, m.LoginEmail, to, msg)
	if err != nil {
		return err
	}

	return nil
}
