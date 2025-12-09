package infrastructure

import (
	"log"
	"net/smtp"
	"os"
)

type EmailService interface {
	SendVerificationCode(to, code string) error
}

type smtpEmailService struct {
	host     string
	port     string
	username string
	password string
	from     string
}

func NewEmailService(host, port, username, password, from string) EmailService {
	return &smtpEmailService{
		host:     host,
		port:     port,
		username: username,
		password: password,
		from:     from,
	}
}

func (s *smtpEmailService) SendVerificationCode(to, code string) error {
	if s.host == "" {
		log.Printf("📧 [MOCK EMAIL] To: %s | Code: %s", to, code)
		return nil
	}

	var auth smtp.Auth
	if s.username != "" {
		auth = smtp.PlainAuth("", s.username, s.password, s.host)
	}

	msg := []byte("To: " + to + "\r\n" +
		"Subject: Verification Code\r\n" +
		"\r\n" +
		"Your verification code is: " + code + "\r\n")

	addr := s.host + ":" + s.port
	return smtp.SendMail(addr, auth, s.from, []string{to}, msg)
}

func NewEmailServiceFromEnv() EmailService {
	return NewEmailService(
		os.Getenv("SMTP_HOST"),
		os.Getenv("SMTP_PORT"),
		os.Getenv("SMTP_USER"),
		os.Getenv("SMTP_PASS"),
		os.Getenv("SMTP_FROM"),
	)
}

