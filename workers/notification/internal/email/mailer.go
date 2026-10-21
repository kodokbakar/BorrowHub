package email

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/smtp"
	"strings"

	"github.com/google/uuid"
)

type Email struct {
	To      string
	Subject string
	Body    string
}

type Mailer interface {
	Send(ctx context.Context, email Email) error
}

type SMTPMailer struct {
	host string
	port string
	user string
	pass string
	from string
}

func NewSMTPMailer(host, port, user, pass, from string) *SMTPMailer {
	return &SMTPMailer{
		host: host,
		port: port,
		user: user,
		pass: pass,
		from: from,
	}
}

func (m *SMTPMailer) Send(ctx context.Context, email Email) error {
	if email.To == "" {
		return errors.New("recipient email is required")
	}
	if ctx.Err() != nil {
		return ctx.Err()
	}

	addr := net.JoinHostPort(m.host, m.port)
	var auth smtp.Auth
	if m.user != "" && m.pass != "" {
		auth = smtp.PlainAuth("", m.user, m.pass, m.host)
	}

	// RFC822 message format
	msg := []byte(fmt.Sprintf(
		"From: %s\r\n"+
			"To: %s\r\n"+
			"Subject: %s\r\n"+
			"MIME-Version: 1.0\r\n"+
			"Content-Type: text/plain; charset=UTF-8\r\n"+
			"\r\n"+
			"%s\r\n",
		m.from, email.To, email.Subject, email.Body,
	))

	done := make(chan error, 1)
	go func() {
		done <- smtp.SendMail(addr, auth, m.from, []string{email.To}, msg)
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-done:
		if err != nil {
			return fmt.Errorf("sending email via smtp: %w", err)
		}
		return nil
	}
}

type NotificationContent struct {
	Title           string
	InAppMessage    string
	EmailSubject    string
	EmailBody       string
	RecipientUserID uuid.UUID
	RecipientEmail  string
}

// BuildNotificationContent generates in-app and email texts for a given domain event.
func BuildNotificationContent(eventType string, data map[string]any) (*NotificationContent, error) {
	if data == nil {
		return nil, errors.New("missing event data")
	}

	var recipientStr string
	switch eventType {
	case "loan.requested":
		recipientStr, _ = data["requester_user_id"].(string)
	case "loan.approved", "loan.checked_out", "loan.overdue", "loan.returned":
		recipientStr, _ = data["borrower_user_id"].(string)
	case "loan.cancelled":
		recipientStr, _ = data["user_id"].(string)
	}

	if recipientStr == "" {
		// Attempt fallback keys
		if id, ok := data["recipient_user_id"].(string); ok && id != "" {
			recipientStr = id
		} else if id, ok := data["borrower_user_id"].(string); ok && id != "" {
			recipientStr = id
		} else if id, ok := data["requester_user_id"].(string); ok && id != "" {
			recipientStr = id
		} else if id, ok := data["user_id"].(string); ok && id != "" {
			recipientStr = id
		}
	}

	recipientUID, err := uuid.Parse(recipientStr)
	if err != nil {
		return nil, fmt.Errorf("invalid or missing recipient user ID in event data: %w", err)
	}

	var recipientEmail string
	if em, ok := data["email"].(string); ok && em != "" {
		recipientEmail = em
	} else if em, ok := data["borrower_email"].(string); ok && em != "" {
		recipientEmail = em
	} else if em, ok := data["requester_email"].(string); ok && em != "" {
		recipientEmail = em
	} else {
		recipientEmail = fmt.Sprintf("%s@borrowhub.local", recipientUID.String())
	}

	content := &NotificationContent{
		RecipientUserID: recipientUID,
		RecipientEmail:  recipientEmail,
	}

	switch eventType {
	case "loan.requested":
		itemID := data["item_id"]
		qty := data["quantity"]
		purpose := data["purpose"]
		content.Title = "Loan Request Submitted"
		content.InAppMessage = fmt.Sprintf("Your loan request for item %v has been submitted.", itemID)
		content.EmailSubject = "Loan Request Received - BorrowHub"
		content.EmailBody = fmt.Sprintf(
			"Hello,\n\nYour loan request for item %v (quantity: %v) has been received and is pending review.\nPurpose: %v\n\nThank you,\nBorrowHub Team",
			itemID, qty, purpose,
		)

	case "loan.approved":
		loanID := data["loan_id"]
		content.Title = "Loan Request Approved"
		content.InAppMessage = fmt.Sprintf("Your loan request has been approved! Loan ID: %v.", loanID)
		content.EmailSubject = "Loan Request Approved - BorrowHub"
		content.EmailBody = fmt.Sprintf(
			"Hello,\n\nYour loan request has been approved! Loan ID: %v.\nYou may now proceed with checkout.\n\nThank you,\nBorrowHub Team",
			loanID,
		)

	case "loan.checked_out":
		loanID := data["loan_id"]
		checkoutTime := data["checkout_time"]
		content.Title = "Item Checked Out"
		content.InAppMessage = fmt.Sprintf("Your loan %v has been checked out.", loanID)
		content.EmailSubject = "Item Checked Out - BorrowHub"
		content.EmailBody = fmt.Sprintf(
			"Hello,\n\nYour loan %v was checked out at %v.\nPlease remember to return the item on time.\n\nThank you,\nBorrowHub Team",
			loanID, checkoutTime,
		)

	case "loan.overdue":
		loanID := data["loan_id"]
		expectedReturn := data["expected_return_time"]
		content.Title = "Loan Overdue Notice"
		content.InAppMessage = fmt.Sprintf("Your loan %v is overdue! Expected return: %v.", loanID, expectedReturn)
		content.EmailSubject = "URGENT: Loan Overdue Notice - BorrowHub"
		content.EmailBody = fmt.Sprintf(
			"Hello,\n\nYour loan %v was due on %v and is now OVERDUE.\nPlease return the item immediately to avoid penalties.\n\nThank you,\nBorrowHub Team",
			loanID, expectedReturn,
		)

	case "loan.returned":
		loanID := data["loan_id"]
		condition := data["return_condition"]
		content.Title = "Item Returned Successfully"
		content.InAppMessage = fmt.Sprintf("Your loan %v has been returned. Condition: %v.", loanID, condition)
		content.EmailSubject = "Item Return Confirmation - BorrowHub"
		content.EmailBody = fmt.Sprintf(
			"Hello,\n\nYour loan %v has been successfully returned.\nReturn condition: %v\n\nThank you,\nBorrowHub Team",
			loanID, condition,
		)
	case "loan.cancelled":
		loanReqID := data["loan_request_id"]
		reason := data["reason"]
		content.Title = "Loan Request Cancelled"
		if reason != nil && fmt.Sprintf("%v", reason) != "" {
			content.InAppMessage = fmt.Sprintf("Your loan request %v has been cancelled. Reason: %v.", loanReqID, reason)
			content.EmailSubject = "Loan Request Cancelled - BorrowHub"
			content.EmailBody = fmt.Sprintf(
				"Hello,\n\nYour loan request %v has been cancelled.\nReason: %v\n\nThank you,\nBorrowHub Team",
				loanReqID, reason,
			)
		} else {
			content.InAppMessage = fmt.Sprintf("Your loan request %v has been cancelled.", loanReqID)
			content.EmailSubject = "Loan Request Cancelled - BorrowHub"
			content.EmailBody = fmt.Sprintf(
				"Hello,\n\nYour loan request %v has been cancelled.\n\nThank you,\nBorrowHub Team",
				loanReqID,
			)
		}

	case "loan.rejected":
		loanReqID := data["loan_request_id"]
		notes := data["review_notes"]
		content.Title = "Loan Request Rejected"
		if notes != nil && fmt.Sprintf("%v", notes) != "" {
			content.InAppMessage = fmt.Sprintf("Your loan request %v was rejected. Notes: %v.", loanReqID, notes)
			content.EmailSubject = "Loan Request Rejected - BorrowHub"
			content.EmailBody = fmt.Sprintf(
				"Hello,\n\nYour loan request %v has been rejected.\nReview notes: %v\n\nThank you,\nBorrowHub Team",
				loanReqID, notes,
			)
		} else {
			content.InAppMessage = fmt.Sprintf("Your loan request %v was rejected.", loanReqID)
			content.EmailSubject = "Loan Request Rejected - BorrowHub"
			content.EmailBody = fmt.Sprintf(
				"Hello,\n\nYour loan request %v has been rejected.\n\nThank you,\nBorrowHub Team",
				loanReqID,
			)
		}

	default:
		content.Title = fmt.Sprintf("Notification: %s", eventType)
		content.InAppMessage = fmt.Sprintf("You have a new update for event: %s", eventType)
		content.EmailSubject = fmt.Sprintf("Update: %s - BorrowHub", eventType)
		content.EmailBody = fmt.Sprintf("Hello,\n\nYou have received an update for %s.\n\nBorrowHub Team", eventType)
	}
	// Sanitize output strings
	content.Title = strings.TrimSpace(content.Title)
	content.InAppMessage = strings.TrimSpace(content.InAppMessage)
	content.EmailSubject = strings.TrimSpace(content.EmailSubject)
	content.EmailBody = strings.TrimSpace(content.EmailBody)

	return content, nil
}
