package email

import (
	"errors"
	"fmt"
	"net"
	"net/mail"
	"net/smtp"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	SMTPHost string
	SMTPPort int
	Username string
	Password string
	From     string
	To       string
}

func (c Config) Validate() error {
	if c.SMTPHost == "" {
		return errors.New("smtp host is required")
	}
	if c.SMTPPort <= 0 || c.SMTPPort > 65535 {
		return errors.New("smtp port must be between 1 and 65535")
	}
	if _, err := mail.ParseAddress(c.From); err != nil {
		return fmt.Errorf("from address: %w", err)
	}
	if _, err := mail.ParseAddress(c.To); err != nil {
		return fmt.Errorf("to address: %w", err)
	}
	return nil
}

// Send delivers a plain-text notification email over SMTP using the
// caller-provided server. subject is always a fixed, code-controlled
// string; the caller-supplied requester data belongs in body, never in
// subject/from/to, so it can never be interpreted as a header.
func Send(cfg Config, subject, body string) error {
	if err := cfg.Validate(); err != nil {
		return err
	}

	var auth smtp.Auth
	if cfg.Username != "" {
		auth = smtp.PlainAuth("", cfg.Username, cfg.Password, cfg.SMTPHost)
	}

	addr := net.JoinHostPort(cfg.SMTPHost, strconv.Itoa(cfg.SMTPPort))
	msg := buildMessage(cfg.From, cfg.To, subject, body)
	return smtp.SendMail(addr, auth, cfg.From, []string{cfg.To}, msg)
}

func buildMessage(from, to, subject, body string) []byte {
	var b strings.Builder
	fmt.Fprintf(&b, "From: %s\r\n", from)
	fmt.Fprintf(&b, "To: %s\r\n", to)
	fmt.Fprintf(&b, "Subject: %s\r\n", subject)
	fmt.Fprintf(&b, "Date: %s\r\n", time.Now().Format(time.RFC1123Z))
	b.WriteString("Content-Type: text/plain; charset=utf-8\r\n")
	b.WriteString("\r\n")
	b.WriteString(body)
	return []byte(b.String())
}
