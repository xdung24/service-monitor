package monitor

import (
	"bufio"
	"context"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"net"
	"strings"

	"github.com/xdung24/service-monitor/internal/models"
)

// SMTPChecker checks an SMTP server by performing a minimal protocol conversation:
// TCP connect → 220 greeting → EHLO → optional STARTTLS → optional AUTH PLAIN → QUIT.
type SMTPChecker struct{}

// Check performs an SMTP connectivity and (optionally) authentication check.
//
// The check proceeds through these stages:
//  1. TCP connect to m.URL (host:port)
//  2. Optionally wrap the connection in TLS for implicit TLS / SMTPS (m.SMTPUseTLS)
//  3. Read the 220 greeting
//  4. Send EHLO
//  5. If the server advertises STARTTLS and m.SMTPUseTLS is false, upgrade to TLS
//  6. If m.SMTPUsername is set, attempt AUTH PLAIN
//  7. Send QUIT and return UP
func (c *SMTPChecker) Check(ctx context.Context, m *models.Monitor) Result {
	d := dialerFor(m)

	tcpConn, err := d.DialContext(ctx, "tcp", m.URL)
	if err != nil {
		return Result{Status: 0, Message: err.Error()}
	}
	// Use a mutable interface so we can rebind to the TLS wrapper after STARTTLS.
	var conn net.Conn = tcpConn
	defer func() { conn.Close() }() //nolint:errcheck

	// Propagate context deadline to all subsequent I/O on the connection.
	if deadline, ok := ctx.Deadline(); ok {
		conn.SetDeadline(deadline) //nolint:errcheck
	}

	// Implicit TLS (SMTPS, typically port 465): wrap before any SMTP conversation.
	if m.SMTPUseTLS {
		host, _, _ := net.SplitHostPort(m.URL)
		tlsConn := tls.Client(conn, &tls.Config{
			ServerName:         host,
			InsecureSkipVerify: m.SMTPIgnoreTLS, // #nosec G402 -- user opt-in
		})
		if err := tlsConn.Handshake(); err != nil {
			return Result{Status: 0, Message: fmt.Sprintf("TLS handshake: %v", err)}
		}
		conn = tlsConn
	}

	r := bufio.NewReader(conn)

	// Stage 1: Read 220 greeting (may be multi-line).
	greeting, err := readSMTPResponse(r)
	if err != nil {
		return Result{Status: 0, Message: fmt.Sprintf("read greeting: %v", err)}
	}
	if !strings.HasPrefix(greeting[0], "220") {
		return Result{Status: 0, Message: fmt.Sprintf("unexpected greeting: %s", greeting[0])}
	}

	// Stage 2: EHLO.
	if _, err := fmt.Fprintf(conn, "EHLO localhost\r\n"); err != nil {
		return Result{Status: 0, Message: fmt.Sprintf("send EHLO: %v", err)}
	}
	caps, err := readSMTPResponse(r)
	if err != nil {
		return Result{Status: 0, Message: fmt.Sprintf("read EHLO response: %v", err)}
	}
	if !strings.HasPrefix(caps[0], "250") {
		return Result{Status: 0, Message: fmt.Sprintf("EHLO rejected: %s", caps[0])}
	}

	// Stage 3: STARTTLS — upgrade when the server advertises the capability and
	// we are not already running implicit TLS.
	if !m.SMTPUseTLS && smtpHasCap(caps, "STARTTLS") {
		if _, err := fmt.Fprintf(conn, "STARTTLS\r\n"); err != nil {
			return Result{Status: 0, Message: fmt.Sprintf("send STARTTLS: %v", err)}
		}
		resp, err := readSMTPResponse(r)
		if err != nil {
			return Result{Status: 0, Message: fmt.Sprintf("read STARTTLS response: %v", err)}
		}
		if !strings.HasPrefix(resp[0], "220") {
			return Result{Status: 0, Message: fmt.Sprintf("STARTTLS rejected: %s", resp[0])}
		}
		host, _, _ := net.SplitHostPort(m.URL)
		tlsConn := tls.Client(conn, &tls.Config{
			ServerName:         host,
			InsecureSkipVerify: m.SMTPIgnoreTLS, // #nosec G402 -- user opt-in
		})
		if err := tlsConn.Handshake(); err != nil {
			return Result{Status: 0, Message: fmt.Sprintf("TLS handshake after STARTTLS: %v", err)}
		}
		conn = tlsConn
		r = bufio.NewReader(conn)

		// Re-EHLO after TLS upgrade (required by RFC 3207).
		if _, err := fmt.Fprintf(conn, "EHLO localhost\r\n"); err != nil {
			return Result{Status: 0, Message: fmt.Sprintf("send EHLO (post-TLS): %v", err)}
		}
		caps, err = readSMTPResponse(r)
		if err != nil {
			return Result{Status: 0, Message: fmt.Sprintf("read EHLO (post-TLS): %v", err)}
		}
		if !strings.HasPrefix(caps[0], "250") {
			return Result{Status: 0, Message: fmt.Sprintf("EHLO (post-TLS) rejected: %s", caps[0])}
		}
	}

	// Stage 4: Optional AUTH PLAIN — verify credentials when configured.
	if m.SMTPUsername != "" {
		creds := base64.StdEncoding.EncodeToString(
			[]byte("\x00" + m.SMTPUsername + "\x00" + m.SMTPPassword),
		)
		if _, err := fmt.Fprintf(conn, "AUTH PLAIN %s\r\n", creds); err != nil {
			return Result{Status: 0, Message: fmt.Sprintf("send AUTH: %v", err)}
		}
		resp, err := readSMTPResponse(r)
		if err != nil {
			return Result{Status: 0, Message: fmt.Sprintf("read AUTH response: %v", err)}
		}
		// 235 = authentication successful.
		if !strings.HasPrefix(resp[0], "235") {
			return Result{Status: 0, Message: fmt.Sprintf("AUTH failed: %s", resp[0])}
		}
	}

	// Graceful QUIT (best-effort; we're about to return UP regardless).
	fmt.Fprintf(conn, "QUIT\r\n") //nolint:errcheck
	return Result{Status: 1, Message: "SMTP OK"}
}

// readSMTPResponse reads a potentially multi-line SMTP response.
// Continuation lines have the form "CODE-text\r\n"; the final line has "CODE text\r\n".
func readSMTPResponse(r *bufio.Reader) ([]string, error) {
	var lines []string
	for {
		line, err := r.ReadString('\n')
		if err != nil {
			return lines, err
		}
		line = strings.TrimRight(line, "\r\n")
		lines = append(lines, line)
		// A line shorter than 4 chars or whose 4th character is not '-' is the final line.
		if len(line) < 4 || line[3] != '-' {
			break
		}
	}
	return lines, nil
}

// smtpHasCap reports whether the given capability keyword appears in any EHLO response line.
func smtpHasCap(caps []string, keyword string) bool {
	upper := strings.ToUpper(keyword)
	for _, c := range caps {
		if strings.Contains(strings.ToUpper(c), upper) {
			return true
		}
	}
	return false
}
