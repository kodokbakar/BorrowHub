package email

import (
	"context"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestBuildNotificationContent(t *testing.T) {
	borrowerID := uuid.New()
	requesterID := uuid.New()
	loanID := uuid.New()
	itemID := uuid.New()

	// 1. loan.requested
	reqData := map[string]any{
		"loan_request_id":   uuid.New().String(),
		"item_id":           itemID.String(),
		"requester_user_id": requesterID.String(),
		"quantity":          2,
		"purpose":           "Conference demo",
		"email":             "requester@example.com",
	}
	content, err := BuildNotificationContent("loan.requested", reqData)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if content.RecipientUserID != requesterID {
		t.Fatalf("expected recipient %s, got %s", requesterID, content.RecipientUserID)
	}
	if content.RecipientEmail != "requester@example.com" {
		t.Fatalf("expected email requester@example.com, got %s", content.RecipientEmail)
	}
	if !strings.Contains(content.InAppMessage, itemID.String()) {
		t.Fatalf("expected in-app message to contain item ID, got: %s", content.InAppMessage)
	}

	// 2. loan.approved
	appData := map[string]any{
		"loan_id":          loanID.String(),
		"borrower_user_id": borrowerID.String(),
	}
	content, err = BuildNotificationContent("loan.approved", appData)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if content.RecipientUserID != borrowerID {
		t.Fatalf("expected borrower ID, got %s", content.RecipientUserID)
	}
	if content.RecipientEmail != borrowerID.String()+"@borrowhub.local" {
		t.Fatalf("expected default email fallback, got: %s", content.RecipientEmail)
	}
	if !strings.Contains(content.EmailSubject, "Approved") {
		t.Fatalf("expected subject to contain Approved, got: %s", content.EmailSubject)
	}

	// 3. loan.checked_out
	coData := map[string]any{
		"loan_id":          loanID.String(),
		"borrower_user_id": borrowerID.String(),
		"checkout_time":    "2026-10-21T10:00:00Z",
	}
	content, err = BuildNotificationContent("loan.checked_out", coData)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(content.InAppMessage, "checked out") {
		t.Fatalf("expected checked out message, got: %s", content.InAppMessage)
	}

	// 4. loan.overdue
	odData := map[string]any{
		"loan_id":              loanID.String(),
		"borrower_user_id":     borrowerID.String(),
		"expected_return_time": "2026-10-20T10:00:00Z",
	}
	content, err = BuildNotificationContent("loan.overdue", odData)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(content.EmailSubject, "Overdue") {
		t.Fatalf("expected Overdue in subject, got: %s", content.EmailSubject)
	}

	// 5. loan.returned
	retData := map[string]any{
		"loan_id":          loanID.String(),
		"borrower_user_id": borrowerID.String(),
		"return_condition": "good",
	}
	content, err = BuildNotificationContent("loan.returned", retData)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(content.InAppMessage, "good") {
		t.Fatalf("expected condition in message, got: %s", content.InAppMessage)
	}
	// 6. loan.cancelled with user_id
	userID := uuid.New()
	cancelData := map[string]any{
		"loan_request_id": loanID.String(),
		"user_id":         userID.String(),
		"reason":          "User requested cancellation",
	}
	content, err = BuildNotificationContent("loan.cancelled", cancelData)
	if err != nil {
		t.Fatalf("unexpected error for loan.cancelled: %v", err)
	}
	if content.RecipientUserID != userID {
		t.Fatalf("expected recipient %s, got %s", userID, content.RecipientUserID)
	}
	if !strings.Contains(content.InAppMessage, "cancelled") || !strings.Contains(content.InAppMessage, "User requested cancellation") {
		t.Fatalf("expected cancellation message with reason, got: %s", content.InAppMessage)
	}

	// 7. loan.cancelled with user_id but without reason
	cancelDataNoReason := map[string]any{
		"loan_request_id": loanID.String(),
		"user_id":         userID.String(),
	}
	content, err = BuildNotificationContent("loan.cancelled", cancelDataNoReason)
	if err != nil {
		t.Fatalf("unexpected error for loan.cancelled without reason: %v", err)
	}
	if content.RecipientUserID != userID {
		t.Fatalf("expected recipient %s, got %s", userID, content.RecipientUserID)
	}
	if !strings.Contains(content.InAppMessage, "cancelled") {
		t.Fatalf("expected cancellation message, got: %s", content.InAppMessage)
	}

	// 8. Fallback to user_id for generic event
	genericData := map[string]any{
		"user_id": userID.String(),
	}
	content, err = BuildNotificationContent("custom.event", genericData)
	if err != nil {
		t.Fatalf("unexpected error for fallback user_id: %v", err)
	}
	if content.RecipientUserID != userID {
		t.Fatalf("expected recipient %s from user_id fallback, got %s", userID, content.RecipientUserID)
	}

	// 9. Invalid recipient ID
	badData := map[string]any{
		"loan_id": loanID.String(),
	}
	_, err = BuildNotificationContent("loan.approved", badData)
	if err == nil {
		t.Fatal("expected error for missing recipient ID, got nil")
	}
}

func TestSMTPMailer_MockServer(t *testing.T) {
	// Start an in-memory mock SMTP server
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to listen: %v", err)
	}
	defer listener.Close()

	_, port, err := net.SplitHostPort(listener.Addr().String())
	if err != nil {
		t.Fatalf("failed to split host port: %v", err)
	}

	receivedMsg := make(chan string, 1)

	go func() {
		conn, err := listener.Accept()
		if err != nil {
			return
		}
		defer conn.Close()

		// Send greeting
		conn.Write([]byte("220 mock.smtp Service Ready\r\n"))

		buf := make([]byte, 2048)
		var fullMessage strings.Builder

		for {
			n, err := conn.Read(buf)
			if err != nil {
				break
			}
			cmd := string(buf[:n])
			fullMessage.WriteString(cmd)

			if strings.HasPrefix(cmd, "EHLO") || strings.HasPrefix(cmd, "HELO") {
				conn.Write([]byte("250-mock.smtp\r\n250 OK\r\n"))
			} else if strings.HasPrefix(cmd, "MAIL FROM:") {
				conn.Write([]byte("250 Sender OK\r\n"))
			} else if strings.HasPrefix(cmd, "RCPT TO:") {
				conn.Write([]byte("250 Recipient OK\r\n"))
			} else if strings.HasPrefix(cmd, "DATA") {
				conn.Write([]byte("354 Start mail input; end with <CRLF>.<CRLF>\r\n"))
			} else if strings.Contains(cmd, "\r\n.\r\n") || strings.HasSuffix(cmd, ".\r\n") {
				conn.Write([]byte("250 Message accepted for delivery\r\n"))
			} else if strings.HasPrefix(cmd, "QUIT") {
				conn.Write([]byte("221 Bye\r\n"))
				break
			}
		}
		receivedMsg <- fullMessage.String()
	}()

	mailer := NewSMTPMailer("127.0.0.1", port, "", "", "notifications@borrowhub.local")
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	err = mailer.Send(ctx, Email{
		To:      "test@example.com",
		Subject: "Test Subject",
		Body:    "Test Body Message",
	})
	if err != nil {
		t.Fatalf("failed to send email: %v", err)
	}

	select {
	case msg := <-receivedMsg:
		if !strings.Contains(msg, "Subject: Test Subject") {
			t.Fatalf("expected message to contain Subject, got: %s", msg)
		}
		if !strings.Contains(msg, "Test Body Message") {
			t.Fatalf("expected message to contain Body, got: %s", msg)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for email on mock server")
	}
}
