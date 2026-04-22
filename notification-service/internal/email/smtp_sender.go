package email

import (
	"bytes"
	"fmt"
	"html/template"

	"gopkg.in/gomail.v2"
)

// Sender sends emails via SMTP.
type Sender interface {
	SendOTPEmail(to, otpCode string, expiresInSeconds int) error
}

type smtpSender struct {
	dialer *gomail.Dialer
	from   string
}

func NewSMTPSender(host string, port int, username, password, from string) Sender {
	d := gomail.NewDialer(host, port, username, password)
	return &smtpSender{dialer: d, from: from}
}

var otpTemplate = template.Must(template.New("otp").Parse(`<!DOCTYPE html>
<html lang="vi">
<head><meta charset="UTF-8"></head>
<body style="font-family:Arial,sans-serif;max-width:480px;margin:0 auto;padding:24px;">
  <h2 style="color:#1a1a1a;">Xác thực tài khoản của bạn</h2>
  <p>Mã OTP của bạn là:</p>
  <div style="text-align:center;margin:24px 0;">
    <span style="font-size:36px;font-weight:bold;letter-spacing:12px;color:#4F46E5;">{{.OTPCode}}</span>
  </div>
  <p>Mã có hiệu lực trong <strong>{{.ExpiresInMinutes}} phút</strong>.</p>
  <p style="color:#6b7280;font-size:13px;">Không chia sẻ mã này với bất kỳ ai. Nếu bạn không yêu cầu, hãy bỏ qua email này.</p>
</body>
</html>`))

type otpData struct {
	OTPCode          string
	ExpiresInMinutes int
}

func (s *smtpSender) SendOTPEmail(to, otpCode string, expiresInSeconds int) error {
	var buf bytes.Buffer
	data := otpData{
		OTPCode:          otpCode,
		ExpiresInMinutes: expiresInSeconds / 60,
	}
	if err := otpTemplate.Execute(&buf, data); err != nil {
		return fmt.Errorf("smtp: render template: %w", err)
	}

	m := gomail.NewMessage()
	m.SetHeader("From", s.from)
	m.SetHeader("To", to)
	m.SetHeader("Subject", "Mã OTP xác thực tài khoản")
	m.SetBody("text/html", buf.String())

	if err := s.dialer.DialAndSend(m); err != nil {
		return fmt.Errorf("smtp: send: %w", err)
	}
	return nil
}
