package service

import (
	"bytes"
	"html/template"
	"io"
	"net"
	"strings"

	"github.com/emersion/go-sasl"
	"github.com/emersion/go-smtp"
	i18nc "github.com/oodzchen/dproject/i18n"
)

type Mail struct {
	LoginEmail     string
	auth           sasl.Client
	SMTPServer     string
	SMTPServerPort string
	i18nCustom     *i18nc.I18nCustom
}

type MailHead struct {
	To, From, Subject string
}

type MailBody struct {
	DomainName, Code string
	Minutes          int
}

const mailHeadTpl = `To: {{.To}}
From: {{.From}}
Subject: {{.Subject}}
MIME-version: 1.0;
Content-Type: text/html;charset="UTF-8";
`

// const verificationCodeTpl = `<html>
// <body>
// <p>You are registering on {{.DomainName}}, here's the verfication code:</p>
// <p><large><b>{{.Code}}</b></large></p>
// <p>Valid for {{.Minutes}} minutes.</p>
// <hr>
// <p style="color:#666">{{.DomainName}}</p>
// </body>
// </html>`

func NewMail(userEmail, password string, smtpServer, smtpServerPort string, i18nCustom *i18nc.I18nCustom) *Mail {
	auth := sasl.NewPlainClient("", userEmail, password)

	return &Mail{userEmail, auth, smtpServer, smtpServerPort, i18nCustom}
}

func (m *Mail) SendVerificationCode(email, code string) error {
	addr := net.JoinHostPort(m.SMTPServer, m.SMTPServerPort)

	var buf bytes.Buffer
	writer := io.MultiWriter(&buf)
	headTpl := template.Must(template.New("head").Parse(mailHeadTpl))
	err := headTpl.Execute(writer, &MailHead{
		To:      email,
		From:    m.LoginEmail,
		Subject: m.i18nCustom.LocalTpl("VerificationMailTitle"),
	})
	if err != nil {
		return err
	}

	verificationCodeTpl := m.i18nCustom.LocalTpl("VerificationMailTpl", "DomainName", "dizkaz.com", "Code", code, "Minutes", 5)

	// fmt.Println("verificationCodeTpl: ", verificationCodeTpl)

	bodyTpl := template.Must(template.New("body").Parse(verificationCodeTpl))
	// err = bodyTpl.Execute(writer, &MailBody{
	// 	DomainName: "dizkaz.com",
	// 	Code:       code,
	// 	Minutes:    5,
	// })
	err = bodyTpl.Execute(writer, nil)
	if err != nil {
		return err
	}

	// fmt.Println("msg data: ", buf.String())

	// msg := strings.NewReader(msgData)
	msg := strings.NewReader(buf.String())
	to := []string{email}

	err = smtp.SendMail(addr, m.auth, m.LoginEmail, to, msg)
	// err := smtp.SendMail(email, m.auth, m.LoginEmail, to, msg)
	if err != nil {
		return err
	}

	return nil
}
