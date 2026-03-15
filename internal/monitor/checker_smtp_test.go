package monitor

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/xdung24/service-monitor/internal/models"
)

// smtpTestServer starts a minimal in-process fake SMTP server that accepts ONE
// connection, calls handler, then closes. Returns the server's host:port address.
func smtpTestServer(t *testing.T, handler func(conn net.Conn)) string {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("smtpTestServer: listen: %v", err)
	}
	t.Cleanup(func() { ln.Close() })
	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		handler(conn)
	}()
	return ln.Addr().String()
}

// smtpBaseMonitor returns a minimal monitor configured for SMTP.
func smtpBaseMonitor(addr string) *models.Monitor {
	m := baseMonitor(addr)
	m.Type = models.MonitorTypeSMTP
	return m
}

// ---------------------------------------------------------------------------
// Basic connectivity tests
// ---------------------------------------------------------------------------

func TestSMTPChecker_OK(t *testing.T) {
	addr := smtpTestServer(t, func(conn net.Conn) {
		r := bufio.NewReader(conn)
		fmt.Fprint(conn, "220 smtp.test ESMTP\r\n")
		r.ReadString('\n') // EHLO
		fmt.Fprint(conn, "250 smtp.test\r\n")
		r.ReadString('\n') // QUIT
		fmt.Fprint(conn, "221 Bye\r\n")
	})
	r := (&SMTPChecker{}).Check(context.Background(), smtpBaseMonitor(addr))
	if r.Status != 1 {
		t.Errorf("want UP, got DOWN: %s", r.Message)
	}
	if r.Message != "SMTP OK" {
		t.Errorf("want message 'SMTP OK', got %q", r.Message)
	}
}

func TestSMTPChecker_BadGreeting(t *testing.T) {
	addr := smtpTestServer(t, func(conn net.Conn) {
		fmt.Fprint(conn, "421 Service temporarily unavailable\r\n")
	})
	r := (&SMTPChecker{}).Check(context.Background(), smtpBaseMonitor(addr))
	if r.Status != 0 {
		t.Error("want DOWN for non-220 greeting, got UP")
	}
}

func TestSMTPChecker_EHLOFails(t *testing.T) {
	addr := smtpTestServer(t, func(conn net.Conn) {
		r := bufio.NewReader(conn)
		fmt.Fprint(conn, "220 ok\r\n")
		r.ReadString('\n') // EHLO
		fmt.Fprint(conn, "502 Command not implemented\r\n")
	})
	r := (&SMTPChecker{}).Check(context.Background(), smtpBaseMonitor(addr))
	if r.Status != 0 {
		t.Error("want DOWN when EHLO is rejected, got UP")
	}
}

func TestSMTPChecker_MultiLineGreeting(t *testing.T) {
	addr := smtpTestServer(t, func(conn net.Conn) {
		r := bufio.NewReader(conn)
		fmt.Fprint(conn, "220-smtp.test ESMTP ready\r\n")
		fmt.Fprint(conn, "220 Welcome!\r\n")
		r.ReadString('\n') // EHLO
		fmt.Fprint(conn, "250 smtp.test\r\n")
		r.ReadString('\n') // QUIT
		fmt.Fprint(conn, "221 Bye\r\n")
	})
	r := (&SMTPChecker{}).Check(context.Background(), smtpBaseMonitor(addr))
	if r.Status != 1 {
		t.Errorf("want UP with multi-line greeting, got DOWN: %s", r.Message)
	}
}

func TestSMTPChecker_MultiLineEHLO(t *testing.T) {
	addr := smtpTestServer(t, func(conn net.Conn) {
		r := bufio.NewReader(conn)
		fmt.Fprint(conn, "220 smtp.test ESMTP\r\n")
		r.ReadString('\n') // EHLO
		fmt.Fprint(conn, "250-smtp.test\r\n250-SIZE 10240000\r\n250-8BITMIME\r\n250 SMTPUTF8\r\n")
		r.ReadString('\n') // QUIT
		fmt.Fprint(conn, "221 Bye\r\n")
	})
	r := (&SMTPChecker{}).Check(context.Background(), smtpBaseMonitor(addr))
	if r.Status != 1 {
		t.Errorf("want UP with multi-line EHLO, got DOWN: %s", r.Message)
	}
}

func TestSMTPChecker_ConnRefused(t *testing.T) {
	// Use a port that is very unlikely to have a listener.
	m := smtpBaseMonitor("127.0.0.1:19991")
	r := (&SMTPChecker{}).Check(context.Background(), m)
	if r.Status != 0 {
		t.Error("want DOWN on connection refused, got UP")
	}
}

func TestSMTPChecker_Timeout(t *testing.T) {
	// Server that accepts but never sends the 220 greeting.
	addr := smtpTestServer(t, func(conn net.Conn) {
		buf := make([]byte, 1)
		conn.Read(buf) // block until client disconnects
	})
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	r := (&SMTPChecker{}).Check(ctx, smtpBaseMonitor(addr))
	if r.Status != 0 {
		t.Error("want DOWN on timeout, got UP")
	}
}

// ---------------------------------------------------------------------------
// AUTH tests
// ---------------------------------------------------------------------------

func TestSMTPChecker_AuthSuccess(t *testing.T) {
	addr := smtpTestServer(t, func(conn net.Conn) {
		r := bufio.NewReader(conn)
		fmt.Fprint(conn, "220 smtp.test ESMTP\r\n")
		r.ReadString('\n') // EHLO
		fmt.Fprint(conn, "250-smtp.test\r\n250 AUTH PLAIN LOGIN\r\n")
		r.ReadString('\n') // AUTH PLAIN ...
		fmt.Fprint(conn, "235 Authentication successful\r\n")
		r.ReadString('\n') // QUIT
		fmt.Fprint(conn, "221 Bye\r\n")
	})
	m := smtpBaseMonitor(addr)
	m.SMTPUsername = "user@example.com"
	m.SMTPPassword = "secret"
	r := (&SMTPChecker{}).Check(context.Background(), m)
	if r.Status != 1 {
		t.Errorf("want UP with valid AUTH, got DOWN: %s", r.Message)
	}
}

func TestSMTPChecker_AuthFails(t *testing.T) {
	addr := smtpTestServer(t, func(conn net.Conn) {
		r := bufio.NewReader(conn)
		fmt.Fprint(conn, "220 smtp.test ESMTP\r\n")
		r.ReadString('\n') // EHLO
		fmt.Fprint(conn, "250-smtp.test\r\n250 AUTH PLAIN LOGIN\r\n")
		r.ReadString('\n') // AUTH PLAIN ...
		fmt.Fprint(conn, "535 5.7.8 Authentication credentials invalid\r\n")
	})
	m := smtpBaseMonitor(addr)
	m.SMTPUsername = "user@example.com"
	m.SMTPPassword = "wrongpassword"
	r := (&SMTPChecker{}).Check(context.Background(), m)
	if r.Status != 0 {
		t.Error("want DOWN when AUTH fails, got UP")
	}
}

// ---------------------------------------------------------------------------
// STARTTLS tests (plain TCP upgrade, no real TLS cert needed)
// ---------------------------------------------------------------------------

func TestSMTPChecker_STARTTLSAdvertised_NotOffered(t *testing.T) {
	// Server advertises no STARTTLS — checker skips TLS upgrade and succeeds.
	addr := smtpTestServer(t, func(conn net.Conn) {
		r := bufio.NewReader(conn)
		fmt.Fprint(conn, "220 smtp.test ESMTP\r\n")
		r.ReadString('\n') // EHLO
		fmt.Fprint(conn, "250-smtp.test\r\n250 8BITMIME\r\n")
		r.ReadString('\n') // QUIT
		fmt.Fprint(conn, "221 Bye\r\n")
	})
	r := (&SMTPChecker{}).Check(context.Background(), smtpBaseMonitor(addr))
	if r.Status != 1 {
		t.Errorf("want UP when no STARTTLS advertised, got DOWN: %s", r.Message)
	}
}
