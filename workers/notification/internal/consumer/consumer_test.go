package consumer

import (
	"context"
	"encoding/json"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"borrowhub/workers/notification/internal/domain"
	"borrowhub/workers/notification/internal/email"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	amqp "github.com/rabbitmq/amqp091-go"
)

type mockAcknowledger struct {
	acked    bool
	nacked   bool
	requeue  bool
	multiple bool
	tag      uint64
}

func (m *mockAcknowledger) Ack(tag uint64, multiple bool) error {
	m.acked = true
	m.tag = tag
	m.multiple = multiple
	return nil
}

func (m *mockAcknowledger) Nack(tag uint64, multiple bool, requeue bool) error {
	m.nacked = true
	m.tag = tag
	m.multiple = multiple
	m.requeue = requeue
	return nil
}

func (m *mockAcknowledger) Reject(tag uint64, requeue bool) error {
	return m.Nack(tag, false, requeue)
}

type mockConsumerRepo struct {
	processedEvents map[uuid.UUID]string
	notifications   map[uuid.UUID]*domain.Notification
	deliveries      map[uuid.UUID]*domain.Delivery
}

func newMockConsumerRepo() *mockConsumerRepo {
	return &mockConsumerRepo{
		processedEvents: make(map[uuid.UUID]string),
		notifications:   make(map[uuid.UUID]*domain.Notification),
		deliveries:      make(map[uuid.UUID]*domain.Delivery),
	}
}

func (m *mockConsumerRepo) RunInTx(ctx context.Context, fn func(tx pgx.Tx) error) error {
	return fn(nil)
}

func (m *mockConsumerRepo) RecordProcessedEvent(ctx context.Context, tx pgx.Tx, eventID uuid.UUID, eventType string) (bool, error) {
	if _, ok := m.processedEvents[eventID]; ok {
		return false, nil
	}
	m.processedEvents[eventID] = eventType
	return true, nil
}

func (m *mockConsumerRepo) CreateNotification(ctx context.Context, tx pgx.Tx, n *domain.Notification) error {
	m.notifications[n.ID] = n
	return nil
}

func (m *mockConsumerRepo) CreateDelivery(ctx context.Context, tx pgx.Tx, d *domain.Delivery) error {
	m.deliveries[d.ID] = d
	return nil
}

func (m *mockConsumerRepo) UpdateDeliveryStatus(ctx context.Context, id uuid.UUID, status domain.DeliveryStatus, attemptsCount int, lastError *string) error {
	if d, ok := m.deliveries[id]; ok {
		d.Status = status
		d.AttemptsCount = attemptsCount
		d.LastError = lastError
	}
	return nil
}

type mockMailer struct {
	sentEmails []email.Email
	failNext   bool
}

func (m *mockMailer) Send(ctx context.Context, e email.Email) error {
	if m.failNext {
		return errors.New("smtp connection refused")
	}
	m.sentEmails = append(m.sentEmails, e)
	return nil
}

func TestConsumer_HappyPath(t *testing.T) {
	repo := newMockConsumerRepo()
	mailer := &mockMailer{}
	consumer := NewConsumer("", repo, mailer, nil)
	ctx := context.Background()

	ack := &mockAcknowledger{}
	eventID := uuid.New()
	orgID := uuid.New()
	borrowerID := uuid.New()
	loanID := uuid.New()

	envelope := domain.DomainEnvelope{
		EventID:        eventID.String(),
		EventType:      "loan.approved",
		EventVersion:   1,
		Source:         "lending-service",
		Subject:        loanID.String(),
		OrganizationID: orgID.String(),
		OccurredAt:     time.Now().Format(time.RFC3339),
		CorrelationID:  "corr-123",
		CausationID:    "cause-123",
		Data: map[string]any{
			"loan_id":          loanID.String(),
			"borrower_user_id": borrowerID.String(),
			"email":            "borrower@example.com",
		},
	}
	body, _ := json.Marshal(envelope)

	msg := amqp.Delivery{
		Acknowledger: ack,
		DeliveryTag:  1,
		Body:         body,
	}

	err := consumer.HandleMessage(ctx, msg)
	if err != nil {
		t.Fatalf("handle message: %v", err)
	}

	if !ack.acked {
		t.Fatal("expected message to be acked")
	}
	if ack.nacked {
		t.Fatal("expected message NOT to be nacked")
	}

	// Verify notification persisted
	if len(repo.notifications) != 1 {
		t.Fatalf("expected 1 notification, got %d", len(repo.notifications))
	}

	// Verify email dispatched
	if len(mailer.sentEmails) != 1 {
		t.Fatalf("expected 1 email sent, got %d", len(mailer.sentEmails))
	}
	if mailer.sentEmails[0].To != "borrower@example.com" {
		t.Fatalf("expected email to borrower@example.com, got %s", mailer.sentEmails[0].To)
	}
}

func TestConsumer_DuplicateIdempotency(t *testing.T) {
	repo := newMockConsumerRepo()
	mailer := &mockMailer{}
	consumer := NewConsumer("", repo, mailer, nil)
	ctx := context.Background()

	eventID := uuid.New()
	orgID := uuid.New()
	borrowerID := uuid.New()
	loanID := uuid.New()

	envelope := domain.DomainEnvelope{
		EventID:        eventID.String(),
		EventType:      "loan.approved",
		EventVersion:   1,
		Source:         "lending-service",
		Subject:        loanID.String(),
		OrganizationID: orgID.String(),
		OccurredAt:     time.Now().Format(time.RFC3339),
		CorrelationID:  "corr-123",
		CausationID:    "cause-123",
		Data: map[string]any{
			"loan_id":          loanID.String(),
			"borrower_user_id": borrowerID.String(),
		},
	}
	body, _ := json.Marshal(envelope)

	// First delivery
	ack1 := &mockAcknowledger{}
	msg1 := amqp.Delivery{Acknowledger: ack1, DeliveryTag: 1, Body: body}
	if err := consumer.HandleMessage(ctx, msg1); err != nil {
		t.Fatalf("first delivery failed: %v", err)
	}
	if !ack1.acked {
		t.Fatal("expected first delivery to be acked")
	}

	// Second delivery (duplicate redelivered message)
	ack2 := &mockAcknowledger{}
	msg2 := amqp.Delivery{Acknowledger: ack2, DeliveryTag: 2, Body: body, Redelivered: true}
	if err := consumer.HandleMessage(ctx, msg2); err != nil {
		t.Fatalf("second delivery failed: %v", err)
	}
	if !ack2.acked {
		t.Fatal("expected duplicate message to be acked without error")
	}

	// Zero duplicate side-effects
	if len(repo.notifications) != 1 {
		t.Fatalf("expected exactly 1 notification, got %d", len(repo.notifications))
	}
	if len(mailer.sentEmails) != 1 {
		t.Fatalf("expected exactly 1 email sent, got %d", len(mailer.sentEmails))
	}
}

func TestConsumer_PoisonMalformedJSON(t *testing.T) {
	repo := newMockConsumerRepo()
	mailer := &mockMailer{}
	consumer := NewConsumer("", repo, mailer, nil)
	ctx := context.Background()

	ack := &mockAcknowledger{}
	msg := amqp.Delivery{
		Acknowledger: ack,
		DeliveryTag:  1,
		Body:         []byte("{invalid-json"),
	}

	err := consumer.HandleMessage(ctx, msg)
	if err == nil {
		t.Fatal("expected error for malformed json, got nil")
	}

	if !ack.nacked {
		t.Fatal("expected malformed message to be nacked")
	}
	if ack.requeue {
		t.Fatal("expected requeue=false so poison message routes to DLQ")
	}
}

func TestConsumer_PoisonMalformedPayload(t *testing.T) {
	repo := newMockConsumerRepo()
	mailer := &mockMailer{}
	consumer := NewConsumer("", repo, mailer, nil)
	ctx := context.Background()

	ack := &mockAcknowledger{}
	envelope := domain.DomainEnvelope{
		EventID:        uuid.New().String(),
		EventType:      "loan.approved",
		EventVersion:   1,
		Source:         "lending-service",
		Subject:        "loan-1",
		OrganizationID: uuid.New().String(),
		OccurredAt:     time.Now().Format(time.RFC3339),
		Data:           map[string]any{}, // missing borrower_user_id!
	}
	body, _ := json.Marshal(envelope)

	msg := amqp.Delivery{
		Acknowledger: ack,
		DeliveryTag:  1,
		Body:         body,
	}

	err := consumer.HandleMessage(ctx, msg)
	if err == nil {
		t.Fatal("expected error for payload missing recipient user ID, got nil")
	}

	if !ack.nacked {
		t.Fatal("expected malformed payload to be nacked")
	}
	if ack.requeue {
		t.Fatal("expected requeue=false (DLQ routing)")
	}
}

func TestConsumer_EmailDispatchFailure(t *testing.T) {
	repo := newMockConsumerRepo()
	mailer := &mockMailer{failNext: true} // SMTP failure
	consumer := NewConsumer("", repo, mailer, nil)
	ctx := context.Background()

	ack := &mockAcknowledger{}
	eventID := uuid.New()
	orgID := uuid.New()
	borrowerID := uuid.New()
	loanID := uuid.New()

	envelope := domain.DomainEnvelope{
		EventID:        eventID.String(),
		EventType:      "loan.approved",
		EventVersion:   1,
		Source:         "lending-service",
		Subject:        loanID.String(),
		OrganizationID: orgID.String(),
		OccurredAt:     time.Now().Format(time.RFC3339),
		Data: map[string]any{
			"loan_id":          loanID.String(),
			"borrower_user_id": borrowerID.String(),
		},
	}
	body, _ := json.Marshal(envelope)

	msg := amqp.Delivery{
		Acknowledger: ack,
		DeliveryTag:  1,
		Body:         body,
	}

	err := consumer.HandleMessage(ctx, msg)
	if err != nil {
		t.Fatalf("unexpected error when email fails: %v", err)
	}

	// Message should still be Acked because domain notification is successfully saved
	if !ack.acked {
		t.Fatal("expected message to be acked even if email fails")
	}

	// Verify email delivery record status marked failed
	var emailDelivery *domain.Delivery
	for _, d := range repo.deliveries {
		if d.Channel == domain.DeliveryChannelEmail {
			emailDelivery = d
			break
		}
	}
	if emailDelivery == nil {
		t.Fatal("expected email delivery record")
	}
	if emailDelivery.Status != domain.DeliveryStatusFailed {
		t.Fatalf("expected delivery status failed, got %s", emailDelivery.Status)
	}
	if emailDelivery.LastError == nil || *emailDelivery.LastError != "smtp connection refused" {
		t.Fatalf("expected last_error recorded, got %+v", emailDelivery.LastError)
	}
}

func TestConsumer_IsReady_InitialAndStopped(t *testing.T) {
	repo := newMockConsumerRepo()
	mailer := &mockMailer{}
	c := NewConsumer("", repo, mailer, nil)

	if c.IsReady() {
		t.Fatal("expected IsReady() to be false before start")
	}

	// Stop on unstarted consumer should not panic or hang
	c.Stop()

	if c.IsReady() {
		t.Fatal("expected IsReady() to be false after stop")
	}
}

func TestConsumer_ReconnectionAndReadiness(t *testing.T) {
	repo := newMockConsumerRepo()
	mailer := &mockMailer{}
	c := NewConsumer("", repo, mailer, nil)
	c.reconnectInitialBackoff = 10 * time.Millisecond
	c.reconnectMaxBackoff = 20 * time.Millisecond

	ch1 := make(chan amqp.Delivery, 1)
	ch2 := make(chan amqp.Delivery, 1)

	var callCount atomic.Int32

	c.setupBrokerFn = func() error {
		return nil
	}
	c.consumeBrokerFn = func() (<-chan amqp.Delivery, error) {
		n := callCount.Add(1)
		if n == 1 {
			return ch1, nil
		}
		return ch2, nil
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := c.Start(ctx); err != nil {
		t.Fatalf("start consumer: %v", err)
	}

	if !c.IsReady() {
		t.Fatal("expected consumer to be ready after start")
	}

	// Simulate channel drop
	close(ch1)

	// Wait for consumer to reconnect and use ch2
	deadline := time.Now().Add(2 * time.Second)
	for {
		if callCount.Load() >= 2 && c.IsReady() {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for consumer to reconnect (calls: %d, ready: %v)", callCount.Load(), c.IsReady())
		}
		time.Sleep(10 * time.Millisecond)
	}

	if !c.IsReady() {
		t.Fatal("expected consumer to be ready after reconnect")
	}

	// Stop consumer
	c.Stop()

	if c.IsReady() {
		t.Fatal("expected consumer to NOT be ready after stop")
	}
}

func TestConsumer_ReconnectionCancelContext(t *testing.T) {
	repo := newMockConsumerRepo()
	mailer := &mockMailer{}
	c := NewConsumer("", repo, mailer, nil)
	c.reconnectInitialBackoff = 10 * time.Millisecond
	c.reconnectMaxBackoff = 20 * time.Millisecond

	ch1 := make(chan amqp.Delivery, 1)
	c.setupBrokerFn = func() error {
		return nil
	}
	c.consumeBrokerFn = func() (<-chan amqp.Delivery, error) {
		return ch1, nil
	}

	ctx, cancel := context.WithCancel(context.Background())

	if err := c.Start(ctx); err != nil {
		t.Fatalf("start consumer: %v", err)
	}

	// Simulate channel closure and context cancellation during reconnect
	c.setupBrokerFn = func() error {
		return errors.New("simulated broker down")
	}
	close(ch1)

	// Give it a moment to hit reconnect loop
	time.Sleep(30 * time.Millisecond)
	if c.IsReady() {
		t.Fatal("expected consumer not to be ready while broker is down")
	}

	// Cancel context to abort reconnect loop
	cancel()

	done := make(chan struct{})
	go func() {
		c.Stop()
		close(done)
	}()

	select {
	case <-done:
		// Clean shutdown verified
	case <-time.After(2 * time.Second):
		t.Fatal("consumer did not stop within deadline after context cancel")
	}
}
