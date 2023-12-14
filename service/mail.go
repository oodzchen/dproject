package service

import (
	"bytes"
	"errors"
	"fmt"
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
	SenderMail     string
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

type VerifCodeType string

const (
	VerifCodeRegister      VerifCodeType = "register"
	VerifCodeResetPassword               = "reset_password"
)

var VerifCodeTypeMap = map[VerifCodeType]bool{
	VerifCodeRegister:      true,
	VerifCodeResetPassword: true,
}

func NewMail(userEmail, password, smtpServer, smtpServerPort, senderAddress string, i18nCustom *i18nc.I18nCustom) *Mail {
	auth := sasl.NewPlainClient("", userEmail, password)

	return &Mail{userEmail, senderAddress, auth, smtpServer, smtpServerPort, i18nCustom}
}

func (m *Mail) SendVerificationCode(email, code string, codeType VerifCodeType) error {
	addr := net.JoinHostPort(m.SMTPServer, m.SMTPServerPort)

	var buf bytes.Buffer
	writer := io.MultiWriter(&buf)
	headTpl := template.Must(template.New("head").Parse(mailHeadTpl))

	var verifMailTplId string
	var mailTitleTplId string
	switch codeType {
	case VerifCodeRegister:
		mailTitleTplId = "VerificationMailTitle"
		verifMailTplId = "VerificationMailTpl"
	case VerifCodeResetPassword:
		mailTitleTplId = "VerificationResetPassMailTitle"
		verifMailTplId = "VerificationResetPassMailTpl"
	}

	if mailTitleTplId == "" {
		return errors.New(fmt.Sprintf("no mail title associate with the code type: %s", string(codeType)))
	}

	err := headTpl.Execute(writer, &MailHead{
		To:      email,
		From:    m.SenderMail,
		Subject: m.i18nCustom.LocalTpl(mailTitleTplId),
	})
	if err != nil {
		return err
	}

	if verifMailTplId == "" {
		return errors.New(fmt.Sprintf("no mail template associate with the code type: %s", string(codeType)))
	}

	verificationCodeTpl := m.i18nCustom.LocalTpl(verifMailTplId, "DomainName", "dizkaz.com", "Code", code, "Minutes", 5)

	bodyTpl := template.Must(template.New("body").Parse(verificationCodeTpl))

	err = bodyTpl.Execute(writer, nil)
	if err != nil {
		return err
	}

	// fmt.Println("msg data: ", buf.String())

	// msg := strings.NewReader(msgData)
	msg := strings.NewReader(buf.String())
	to := []string{email}

	err = smtp.SendMail(addr, m.auth, m.SenderMail, to, msg)
	// err := smtp.SendMail(email, m.auth, m.LoginEmail, to, msg)
	if err != nil {
		return err
	}

	return nil
}
