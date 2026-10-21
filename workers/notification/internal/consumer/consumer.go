package consumer

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"borrowhub/workers/notification/internal/domain"
	"borrowhub/workers/notification/internal/email"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	amqp "github.com/rabbitmq/amqp091-go"
)

const (
	ExchangeDomain    = "borrowhub.domain"
	ExchangeDomainDLX = "borrowhub.domain.dlx"
	QueueEvents       = "notification.events"
	QueueEventsDLQ    = "notification.events.dlq"

	MaxRetries = 3
)

type Repository interface {
	RunInTx(ctx context.Context, fn func(tx pgx.Tx) error) error
	RecordProcessedEvent(ctx context.Context, tx pgx.Tx, eventID uuid.UUID, eventType string) (bool, error)
	CreateNotification(ctx context.Context, tx pgx.Tx, n *domain.Notification) error
	CreateDelivery(ctx context.Context, tx pgx.Tx, d *domain.Delivery) error
	UpdateDeliveryStatus(ctx context.Context, id uuid.UUID, status domain.DeliveryStatus, attemptsCount int, lastError *string) error
}

type Consumer struct {
	rabbitmqURL string
	repo        Repository
	mailer      email.Mailer
	logger      *slog.Logger

	conn     *amqp.Connection
	channel  *amqp.Channel
	mu       sync.Mutex
	stopCh   chan struct{}
	stopped  chan struct{}
	stopOnce sync.Once
	started  atomic.Bool
	ready    atomic.Bool

	reconnectInitialBackoff time.Duration
	reconnectMaxBackoff     time.Duration

	setupBrokerFn   func() error
	consumeBrokerFn func() (<-chan amqp.Delivery, error)
}

func NewConsumer(rabbitmqURL string, repo Repository, mailer email.Mailer, logger *slog.Logger) *Consumer {
	if logger == nil {
		logger = slog.Default()
	}
	return &Consumer{
		rabbitmqURL: rabbitmqURL,
		repo:        repo,
		mailer:      mailer,
		logger:      logger,
		stopCh:      make(chan struct{}),
		stopped:     make(chan struct{}),
	}
}

// IsReady returns true when consumer is actively connected and consuming messages.
func (c *Consumer) IsReady() bool {
	return c.ready.Load()
}

// Start connects to RabbitMQ, sets up topology, and consumes messages until context is cancelled or Stop() is called.
func (c *Consumer) Start(ctx context.Context) error {
	if c.started.Load() {
		return errors.New("consumer already started")
	}

	if err := c.setup(); err != nil {
		return fmt.Errorf("setting up broker: %w", err)
	}

	msgs, err := c.consume()
	if err != nil {
		c.closeBroker()
		return fmt.Errorf("starting consumer: %w", err)
	}

	c.started.Store(true)
	c.ready.Store(true)

	c.logger.Info("notification consumer started", "queue", QueueEvents)

	go c.consumeLoop(ctx, msgs)

	return nil
}

func (c *Consumer) consumeLoop(ctx context.Context, initialMsgs <-chan amqp.Delivery) {
	defer close(c.stopped)
	defer c.ready.Store(false)
	defer c.closeBroker()

	msgs := initialMsgs

	for {
		if msgs == nil {
			var err error
			msgs, err = c.reconnect(ctx)
			if err != nil {
				return
			}
		}

		select {
		case <-ctx.Done():
			c.logger.Info("context cancelled, shutting down consumer")
			return
		case <-c.stopCh:
			c.logger.Info("stop signal received, shutting down consumer")
			return
		case msg, ok := <-msgs:
			if !ok {
				c.logger.Warn("consumer channel closed, initiating reconnect")
				c.ready.Store(false)
				c.closeBroker()
				msgs = nil
				continue
			}
			if err := c.HandleMessage(ctx, msg); err != nil {
				c.logger.Error("error handling message", "error", err)
			}
		}
	}
}

func (c *Consumer) reconnect(ctx context.Context) (<-chan amqp.Delivery, error) {
	backoff := c.reconnectInitialBackoff
	if backoff <= 0 {
		backoff = 500 * time.Millisecond
	}
	maxBackoff := c.reconnectMaxBackoff
	if maxBackoff <= 0 {
		maxBackoff = 30 * time.Second
	}

	for {
		select {
		case <-ctx.Done():
			c.logger.Info("context cancelled during reconnect")
			return nil, ctx.Err()
		case <-c.stopCh:
			c.logger.Info("stop signal received during reconnect")
			return nil, errors.New("consumer stopped")
		case <-time.After(backoff):
		}

		c.logger.Info("attempting to reconnect to rabbitmq")
		if err := c.setup(); err != nil {
			c.logger.Warn("failed to reconnect to rabbitmq", "error", err, "next_retry", backoff)
			if backoff < maxBackoff {
				backoff *= 2
				if backoff > maxBackoff {
					backoff = maxBackoff
				}
			}
			continue
		}

		msgs, err := c.consume()
		if err != nil {
			c.logger.Warn("failed to start consuming after reconnect", "error", err)
			c.closeBroker()
			if backoff < maxBackoff {
				backoff *= 2
				if backoff > maxBackoff {
					backoff = maxBackoff
				}
			}
			continue
		}

		c.ready.Store(true)
		c.logger.Info("successfully reconnected and consuming from rabbitmq", "queue", QueueEvents)
		return msgs, nil
	}
}

func (c *Consumer) Stop() {
	if !c.started.Load() {
		return
	}
	c.stopOnce.Do(func() {
		close(c.stopCh)
	})
	<-c.stopped
}

func (c *Consumer) setup() error {
	if c.setupBrokerFn != nil {
		return c.setupBrokerFn()
	}
	return c.setupBroker()
}

func (c *Consumer) consume() (<-chan amqp.Delivery, error) {
	if c.consumeBrokerFn != nil {
		return c.consumeBrokerFn()
	}
	c.mu.Lock()
	ch := c.channel
	c.mu.Unlock()
	if ch == nil {
		return nil, errors.New("channel is nil")
	}
	return ch.Consume(
		QueueEvents,
		"notification-worker",
		false, // manual ack
		false, // exclusive
		false, // no-local
		false, // no-wait
		nil,   // args
	)
}

func (c *Consumer) closeBrokerLocked() {
	if c.channel != nil {
		_ = c.channel.Close()
		c.channel = nil
	}
	if c.conn != nil {
		_ = c.conn.Close()
		c.conn = nil
	}
}

func (c *Consumer) closeBroker() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.closeBrokerLocked()
}

func (c *Consumer) setupBroker() error {
	conn, err := amqp.Dial(c.rabbitmqURL)
	if err != nil {
		return fmt.Errorf("dialing rabbitmq: %w", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		_ = conn.Close()
		return fmt.Errorf("opening rabbitmq channel: %w", err)
	}

	cleanup := func() {
		_ = ch.Close()
		_ = conn.Close()
	}

	// 1. Declare DLX (borrowhub.domain.dlx)
	if err := ch.ExchangeDeclare(
		ExchangeDomainDLX,
		"topic",
		true,  // durable
		false, // auto-delete
		false, // internal
		false, // no-wait
		nil,
	); err != nil {
		cleanup()
		return fmt.Errorf("declaring dlx exchange: %w", err)
	}

	// 2. Declare DLQ (notification.events.dlq)
	if _, err := ch.QueueDeclare(
		QueueEventsDLQ,
		true,  // durable
		false, // auto-delete
		false, // exclusive
		false, // no-wait
		nil,
	); err != nil {
		cleanup()
		return fmt.Errorf("declaring dlq: %w", err)
	}

	// 3. Bind DLQ to DLX
	if err := ch.QueueBind(
		QueueEventsDLQ,
		"#",
		ExchangeDomainDLX,
		false,
		nil,
	); err != nil {
		cleanup()
		return fmt.Errorf("binding dlq to dlx: %w", err)
	}

	// 4. Declare Domain Exchange (borrowhub.domain)
	if err := ch.ExchangeDeclare(
		ExchangeDomain,
		"topic",
		true,  // durable
		false, // auto-delete
		false, // internal
		false, // no-wait
		nil,
	); err != nil {
		cleanup()
		return fmt.Errorf("declaring domain exchange: %w", err)
	}

	// 5. Declare Main Queue with dead-letter-exchange configured
	queueArgs := amqp.Table{
		"x-dead-letter-exchange": ExchangeDomainDLX,
	}
	if _, err := ch.QueueDeclare(
		QueueEvents,
		true,  // durable
		false, // auto-delete
		false, // exclusive
		false, // no-wait
		queueArgs,
	); err != nil {
		cleanup()
		return fmt.Errorf("declaring events queue: %w", err)
	}

	// 6. Bind Main Queue to domain exchange for loan.*
	if err := ch.QueueBind(
		QueueEvents,
		"loan.*",
		ExchangeDomain,
		false,
		nil,
	); err != nil {
		cleanup()
		return fmt.Errorf("binding queue to loan.*: %w", err)
	}

	// 7. Set prefetch limit
	if err := ch.Qos(20, 0, false); err != nil {
		cleanup()
		return fmt.Errorf("setting qos: %w", err)
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	select {
	case <-c.stopCh:
		cleanup()
		return errors.New("consumer stopped")
	default:
		c.conn = conn
		c.channel = ch
		return nil
	}
}

// HandleMessage processes a single AMQP delivery with idempotency and error handling.
func (c *Consumer) HandleMessage(ctx context.Context, d amqp.Delivery) error {
	// 1. Parse JSON envelope
	var env domain.DomainEnvelope
	if err := json.Unmarshal(d.Body, &env); err != nil {
		c.logger.Error("malformed json envelope, routing to DLQ", "error", err)
		// Poison message: Nack without requeue -> routes to DLX -> DLQ
		_ = d.Nack(false, false)
		return fmt.Errorf("%w: invalid json: %v", domain.ErrMalformedEvent, err)
	}

	// 2. Validate envelope header
	eventID, err := uuid.Parse(env.EventID)
	if err != nil || env.EventType == "" {
		c.logger.Error("invalid envelope header (missing/bad event_id or event_type), routing to DLQ", "event_id", env.EventID, "event_type", env.EventType)
		_ = d.Nack(false, false)
		return fmt.Errorf("%w: missing or invalid event_id/event_type", domain.ErrMalformedEvent)
	}

	orgID, err := uuid.Parse(env.OrganizationID)
	if err != nil {
		c.logger.Error("invalid organization_id, routing to DLQ", "org_id", env.OrganizationID)
		_ = d.Nack(false, false)
		return fmt.Errorf("%w: missing or invalid organization_id", domain.ErrMalformedEvent)
	}

	// 3. Build notification content (validates domain data payload)
	content, err := email.BuildNotificationContent(env.EventType, env.Data)
	if err != nil {
		c.logger.Error("malformed event data, routing to DLQ", "error", err, "event_id", eventID)
		// Domain data missing required fields is a poison message
		_ = d.Nack(false, false)
		return fmt.Errorf("%w: %v", domain.ErrMalformedEvent, err)
	}

	now := time.Now().UTC()
	notifID := uuid.New()
	emailDeliveryID := uuid.New()
	inAppDeliveryID := uuid.New()

	var isNewEvent bool

	// 4. Atomic database transaction with bounded retry for transient DB failures
	var txErr error
	for attempt := 1; attempt <= MaxRetries; attempt++ {
		txErr = c.repo.RunInTx(ctx, func(tx pgx.Tx) error {
			// Atomic deduplication: insert into processed_events ON CONFLICT DO NOTHING
			recorded, rErr := c.repo.RecordProcessedEvent(ctx, tx, eventID, env.EventType)
			if rErr != nil {
				return rErr
			}
			if !recorded {
				// Already processed
				isNewEvent = false
				return nil
			}

			isNewEvent = true

			// Create in-app notification
			notif := &domain.Notification{
				ID:              notifID,
				OrganizationID:  orgID,
				RecipientUserID: content.RecipientUserID,
				EventID:         eventID,
				EventType:       env.EventType,
				Title:           content.Title,
				Message:         content.InAppMessage,
				Data:            env.Data,
				Status:          domain.NotificationStatusUnread,
				CreatedAt:       now,
				UpdatedAt:       now,
			}
			if err := c.repo.CreateNotification(ctx, tx, notif); err != nil {
				return fmt.Errorf("creating notification: %w", err)
			}

			// Create in-app delivery attempt (sent immediately)
			inAppDelivery := &domain.Delivery{
				ID:             inAppDeliveryID,
				NotificationID: notifID,
				Channel:        domain.DeliveryChannelInApp,
				Recipient:      content.RecipientUserID.String(),
				Status:         domain.DeliveryStatusDelivered,
				AttemptsCount:  1,
				CreatedAt:      now,
				UpdatedAt:      now,
			}
			if err := c.repo.CreateDelivery(ctx, tx, inAppDelivery); err != nil {
				return fmt.Errorf("creating in_app delivery: %w", err)
			}

			// Create pending email delivery attempt
			emailDelivery := &domain.Delivery{
				ID:             emailDeliveryID,
				NotificationID: notifID,
				Channel:        domain.DeliveryChannelEmail,
				Recipient:      content.RecipientEmail,
				Status:         domain.DeliveryStatusPending,
				AttemptsCount:  0,
				CreatedAt:      now,
				UpdatedAt:      now,
			}
			if err := c.repo.CreateDelivery(ctx, tx, emailDelivery); err != nil {
				return fmt.Errorf("creating email delivery: %w", err)
			}

			return nil
		})

		if txErr == nil {
			break
		}

		c.logger.Warn("transient database error in consumer, retrying", "attempt", attempt, "error", txErr)
		select {
		case <-ctx.Done():
			_ = d.Nack(false, true) // requeue
			return ctx.Err()
		case <-time.After(time.Duration(attempt*50) * time.Millisecond):
		}
	}

	if txErr != nil {
		c.logger.Error("exhausted retries on database transaction, requeuing message", "error", txErr)
		_ = d.Nack(false, true) // Requeue for broker redelivery
		return txErr
	}

	// If event was duplicate, acknowledge immediately without duplicate side-effects
	if !isNewEvent {
		c.logger.Info("duplicate event ignored", "event_id", eventID)
		_ = d.Ack(false)
		return nil
	}

	// 5. Dispatch email via SMTP mailer
	if c.mailer != nil && content.RecipientEmail != "" {
		sendErr := c.mailer.Send(ctx, email.Email{
			To:      content.RecipientEmail,
			Subject: content.EmailSubject,
			Body:    content.EmailBody,
		})

		if sendErr != nil {
			c.logger.Error("failed to dispatch email", "recipient", content.RecipientEmail, "error", sendErr)
			errStr := sendErr.Error()
			_ = c.repo.UpdateDeliveryStatus(ctx, emailDeliveryID, domain.DeliveryStatusFailed, 1, &errStr)
		} else {
			c.logger.Info("email dispatched successfully", "recipient", content.RecipientEmail)
			_ = c.repo.UpdateDeliveryStatus(ctx, emailDeliveryID, domain.DeliveryStatusDelivered, 1, nil)
		}
	}

	// 6. Acknowledge message successfully processed
	if err := d.Ack(false); err != nil {
		c.logger.Error("failed to ack message", "error", err)
		return err
	}

	return nil
}
