package email

import (
	"bufio"
	"encoding/base64"
	"net"
	"strings"
	"sync"
	"testing"
)

type fakeSMTPServer struct {
	mu       sync.Mutex
	authUser string
	authPass string
	from     string
	to       string
	dataBody string
}

func startFakeSMTPServer(t *testing.T) (host string, port int, srv *fakeSMTPServer) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Listen: %v", err)
	}
	t.Cleanup(func() { ln.Close() })

	srv = &fakeSMTPServer{}
	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		srv.handle(conn)
	}()

	addr := ln.Addr().(*net.TCPAddr)
	return addr.IP.String(), addr.Port, srv
}

func (f *fakeSMTPServer) handle(conn net.Conn) {
	r := bufio.NewReader(conn)
	w := bufio.NewWriter(conn)
	writeLine := func(s string) {
		w.WriteString(s + "\r\n")
		w.Flush()
	}

	writeLine("220 fake.local ESMTP ready")

	for {
		line, err := r.ReadString('\n')
		if err != nil {
			return
		}
		line = strings.TrimRight(line, "\r\n")
		upper := strings.ToUpper(line)

		switch {
		case strings.HasPrefix(upper, "EHLO"):
			w.WriteString("250-fake.local\r\n")
			writeLine("250 AUTH PLAIN")
		case strings.HasPrefix(upper, "HELO"):
			writeLine("250 fake.local")
		case strings.HasPrefix(upper, "AUTH PLAIN"):
			payload := strings.TrimSpace(line[len("AUTH PLAIN"):])
			decoded, err := base64.StdEncoding.DecodeString(payload)
			if err != nil {
				writeLine("501 malformed AUTH PLAIN")
				continue
			}
			parts := strings.Split(string(decoded), "\x00")
			if len(parts) == 3 {
				f.mu.Lock()
				f.authUser = parts[1]
				f.authPass = parts[2]
				f.mu.Unlock()
			}
			writeLine("235 2.7.0 Authentication successful")
		case strings.HasPrefix(upper, "MAIL FROM:"):
			f.mu.Lock()
			f.from = line[len("MAIL FROM:"):]
			f.mu.Unlock()
			writeLine("250 2.1.0 OK")
		case strings.HasPrefix(upper, "RCPT TO:"):
			f.mu.Lock()
			f.to = line[len("RCPT TO:"):]
			f.mu.Unlock()
			writeLine("250 2.1.5 OK")
		case strings.HasPrefix(upper, "DATA"):
			writeLine("354 go ahead")
			var body strings.Builder
			for {
				dataLine, err := r.ReadString('\n')
				if err != nil {
					return
				}
				trimmed := strings.TrimRight(dataLine, "\r\n")
				if trimmed == "." {
					break
				}
				body.WriteString(strings.TrimPrefix(trimmed, "."))
				body.WriteString("\n")
			}
			f.mu.Lock()
			f.dataBody = body.String()
			f.mu.Unlock()
			writeLine("250 2.0.0 OK")
		case strings.HasPrefix(upper, "QUIT"):
			writeLine("221 2.0.0 Bye")
			return
		default:
			writeLine("500 unrecognized command")
		}
	}
}

func (f *fakeSMTPServer) snapshot() (user, pass, from, to, body string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.authUser, f.authPass, f.from, f.to, f.dataBody
}

func TestSendDeliversMessageWithAuth(t *testing.T) {
	host, port, srv := startFakeSMTPServer(t)

	cfg := Config{
		SMTPHost: host,
		SMTPPort: port,
		Username: "shelter@example.com",
		Password: "app-password",
		From:     "shelter@example.com",
		To:       "oncall@example.com",
	}

	if err := Send(cfg, "New intake request", "Jane Doe needs a bed tonight."); err != nil {
		t.Fatalf("Send: %v", err)
	}

	user, pass, from, to, body := srv.snapshot()
	if user != cfg.Username {
		t.Errorf("authUser = %q, want %q", user, cfg.Username)
	}
	if pass != cfg.Password {
		t.Errorf("authPass = %q, want %q", pass, cfg.Password)
	}
	if !strings.Contains(from, cfg.From) {
		t.Errorf("MAIL FROM = %q, want to contain %q", from, cfg.From)
	}
	if !strings.Contains(to, cfg.To) {
		t.Errorf("RCPT TO = %q, want to contain %q", to, cfg.To)
	}
	if !strings.Contains(body, "Jane Doe needs a bed tonight.") {
		t.Errorf("DATA body = %q, want to contain message text", body)
	}
	if !strings.Contains(body, "Subject: New intake request") {
		t.Errorf("DATA body = %q, want a Subject header", body)
	}
}

func TestSendRejectsIncompleteConfig(t *testing.T) {
	cases := []Config{
		{},
		{SMTPHost: "smtp.example.com", SMTPPort: 587, From: "a@example.com"},
		{SMTPHost: "smtp.example.com", SMTPPort: 587, From: "not-an-email", To: "b@example.com"},
		{SMTPHost: "smtp.example.com", SMTPPort: 587, From: "a@example.com", To: "not-an-email"},
	}
	for _, cfg := range cases {
		if err := Send(cfg, "subject", "body"); err == nil {
			t.Errorf("Send(%+v) = nil error, want validation error", cfg)
		}
	}
}

func TestValidateAcceptsCompleteConfig(t *testing.T) {
	cfg := Config{SMTPHost: "smtp.example.com", SMTPPort: 587, From: "a@example.com", To: "b@example.com"}
	if err := cfg.Validate(); err != nil {
		t.Errorf("Validate() = %v, want nil", err)
	}
}

func TestBuildMessageKeepsUntrustedBodyOutOfHeaders(t *testing.T) {
	msg := buildMessage("shelter@example.com", "oncall@example.com", "New intake request", "Name: X\r\nBcc: attacker@evil.com\r\n\r\ninjected")
	got := string(msg)
	headerEnd := strings.Index(got, "\r\n\r\n")
	if headerEnd == -1 {
		t.Fatalf("message has no header/body separator: %q", got)
	}
	headers := got[:headerEnd]
	if strings.Contains(headers, "Bcc:") {
		t.Errorf("attacker-controlled Bcc header leaked into headers section: %q", headers)
	}
}
