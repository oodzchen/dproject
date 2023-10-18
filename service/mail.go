package service

import (
	"bytes"
	"html/template"
	"io"
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

const verificationCodeTpl = `<html>
<body>
<p>You are registering on {{.DomainName}}, here's the verfication code:</p>
<p><large><b>{{.Code}}</b></large></p>
<p>Valid for {{.Minutes}} minutes.</p>
<hr>
<p style="color:#666">{{.DomainName}}</p>
</body>
</html>`

func NewMail(userEmail, password string, smtpServer, smtpServerPort string) *Mail {
	auth := sasl.NewPlainClient("", userEmail, password)

	return &Mail{userEmail, auth, smtpServer, smtpServerPort}
}

func (m *Mail) SendVerificationCode(email, code string) error {
	addr := net.JoinHostPort(m.SMTPServer, m.SMTPServerPort)

	var buf bytes.Buffer
	writer := io.MultiWriter(&buf)
	headTpl := template.Must(template.New("head").Parse(mailHeadTpl))
	err := headTpl.Execute(writer, &MailHead{
		To:      email,
		From:    m.LoginEmail,
		Subject: "Verification code for registration",
	})
	if err != nil {
		return err
	}

	bodyTpl := template.Must(template.New("body").Parse(verificationCodeTpl))
	err = bodyTpl.Execute(writer, &MailBody{
		DomainName: "dizkaz.com",
		Code:       code,
		Minutes:    5,
	})
	if err != nil {
		return err
	}

	// msgTpl := msgHead
	// msgTpl += "<html><body><h1>Verification code</h1>"
	// msgTpl += "The code is <b>%s</b>\r\n<br>"
	// msgTpl += "For security reasons, please do not disclose the verification code.\r\n<br>"
	// msgTpl += "<hr>Kholin\r\n<br>"
	// msgTpl += "</body></html>"
	// msgData := fmt.Sprintf(msgTpl, email, m.LoginEmail, code)

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
