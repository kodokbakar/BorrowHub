package event

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"borrowhub/services/lending/internal/domain"
	"github.com/google/uuid"
	amqp "github.com/rabbitmq/amqp091-go"
)

const (
	ExchangeName = "borrowhub.domain"
	ExchangeType = "topic"
	SourceName   = "lending-service"
	EventVersion = 1

	EventLoanRequested  = "loan.requested"
	EventLoanApproved   = "loan.approved"
	EventLoanRejected   = "loan.rejected"
	EventLoanCancelled  = "loan.cancelled"
	EventLoanCheckedOut = "loan.checked_out"
	EventLoanReturned   = "loan.returned"
	EventLoanOverdue    = "loan.overdue"
)

type DomainEnvelope struct {
	EventID        string         `json:"event_id"`
	EventType      string         `json:"event_type"`
	EventVersion   int            `json:"event_version"`
	Source         string         `json:"source"`
	Subject        string         `json:"subject"`
	OrganizationID string         `json:"organization_id"`
	OccurredAt     string         `json:"occurred_at"`
	CorrelationID  string         `json:"correlation_id"`
	CausationID    string         `json:"causation_id"`
	Data           map[string]any `json:"data"`
}

func BuildOutboxEvent(
	eventType string,
	subject string,
	orgID uuid.UUID,
	correlationID, causationID string,
	data map[string]any,
) domain.OutboxEvent {
	now := time.Now().UTC()
	eventID := uuid.New()
	if correlationID == "" {
		correlationID = eventID.String()
	}
	if causationID == "" {
		causationID = eventID.String()
	}

	return domain.OutboxEvent{
		EventID:        eventID,
		EventType:      eventType,
		EventVersion:   EventVersion,
		Source:         SourceName,
		Subject:        subject,
		OrganizationID: orgID,
		OccurredAt:     now,
		CorrelationID:  correlationID,
		CausationID:    causationID,
		Data:           data,
		Status:         "pending",
		CreatedAt:      now,
	}
}

type OutboxRepository interface {
	ClaimPendingOutboxEvents(ctx context.Context, limit int) ([]domain.OutboxEvent, error)
	GetPendingOutboxEvents(ctx context.Context, limit int) ([]domain.OutboxEvent, error)
	MarkOutboxEventPublished(ctx context.Context, id int64) error
	MarkOutboxEventFailed(ctx context.Context, id int64, errStr string) error
	ResetOutboxEventsToPending(ctx context.Context, ids []int64) error
}

type OutboxPublisher struct {
	repo        OutboxRepository
	rabbitmqURL string
	logger      *slog.Logger

	connMu  sync.RWMutex
	conn    *amqp.Connection
	channel *amqp.Channel

	pollInterval time.Duration
	stopCh       chan struct{}
	wg           sync.WaitGroup
}

func NewOutboxPublisher(
	repo OutboxRepository,
	rabbitmqURL string,
	logger *slog.Logger,
	pollInterval time.Duration,
) *OutboxPublisher {
	if pollInterval <= 0 {
		pollInterval = 500 * time.Millisecond
	}
	if logger == nil {
		logger = slog.Default()
	}

	return &OutboxPublisher{
		repo:         repo,
		rabbitmqURL:  rabbitmqURL,
		logger:       logger.With("component", "outbox_publisher"),
		pollInterval: pollInterval,
		stopCh:       make(chan struct{}),
	}
}

func (p *OutboxPublisher) Start(ctx context.Context) {
	p.wg.Add(1)
	go p.run(ctx)
}

func (p *OutboxPublisher) Stop() {
	close(p.stopCh)
	p.wg.Wait()
	p.closeBroker()
}

func (p *OutboxPublisher) IsHealthy() bool {
	p.connMu.RLock()
	defer p.connMu.RUnlock()
	return p.conn != nil && !p.conn.IsClosed()
}

func (p *OutboxPublisher) run(ctx context.Context) {
	defer p.wg.Done()

	ticker := time.NewTicker(p.pollInterval)
	defer ticker.Stop()

	p.logger.Info("outbox publisher started")

	for {
		select {
		case <-ctx.Done():
			p.logger.Info("outbox publisher shutting down from context")
			return
		case <-p.stopCh:
			p.logger.Info("outbox publisher shutting down from stop signal")
			return
		case <-ticker.C:
			p.processPendingEvents(ctx)
		}
	}
}

func (p *OutboxPublisher) ensureBrokerConnection() error {
	p.connMu.Lock()
	defer p.connMu.Unlock()

	if p.conn != nil && !p.conn.IsClosed() && p.channel != nil {
		return nil
	}

	p.logger.Info("connecting to rabbitmq", "url", p.rabbitmqURL)
	conn, err := amqp.Dial(p.rabbitmqURL)
	if err != nil {
		return fmt.Errorf("connecting to rabbitmq: %w", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return fmt.Errorf("opening rabbitmq channel: %w", err)
	}

	err = ch.ExchangeDeclare(
		ExchangeName,
		ExchangeType,
		true,  // durable
		false, // auto-deleted
		false, // internal
		false, // no-wait
		nil,   // arguments
	)
	if err != nil {
		ch.Close()
		conn.Close()
		return fmt.Errorf("declaring exchange %s: %w", ExchangeName, err)
	}

	p.conn = conn
	p.channel = ch
	p.logger.Info("successfully connected to rabbitmq and declared exchange")
	return nil
}

func (p *OutboxPublisher) closeBroker() {
	p.connMu.Lock()
	defer p.connMu.Unlock()

	if p.channel != nil {
		_ = p.channel.Close()
		p.channel = nil
	}
	if p.conn != nil {
		_ = p.conn.Close()
		p.conn = nil
	}
}

func (p *OutboxPublisher) processPendingEvents(ctx context.Context) {
	if err := p.ensureBrokerConnection(); err != nil {
		p.logger.Warn("broker connection unavailable; outbox events queued", "error", err)
		return
	}

	p.connMu.RLock()
	ch := p.channel
	p.connMu.RUnlock()

	events, err := p.repo.ClaimPendingOutboxEvents(ctx, 50)
	if err != nil {
		p.logger.Error("failed fetching pending outbox events", "error", err)
		return
	}
	if len(events) == 0 {
		return
	}

	for i, ev := range events {
		envelope := DomainEnvelope{
			EventID:        ev.EventID.String(),
			EventType:      ev.EventType,
			EventVersion:   ev.EventVersion,
			Source:         ev.Source,
			Subject:        ev.Subject,
			OrganizationID: ev.OrganizationID.String(),
			OccurredAt:     ev.OccurredAt.UTC().Format(time.RFC3339Nano),
			CorrelationID:  ev.CorrelationID,
			CausationID:    ev.CausationID,
			Data:           ev.Data,
		}

		body, err := json.Marshal(envelope)
		if err != nil {
			p.logger.Error("failed marshaling domain envelope", "event_id", ev.EventID, "error", err)
			_ = p.repo.MarkOutboxEventFailed(ctx, ev.ID, fmt.Sprintf("marshaling: %v", err))
			continue
		}

		pubCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		err = ch.PublishWithContext(
			pubCtx,
			ExchangeName,
			ev.EventType,
			false, // mandatory
			false, // immediate
			amqp.Publishing{
				ContentType:  "application/json",
				DeliveryMode: amqp.Persistent,
				Timestamp:    ev.OccurredAt,
				MessageId:    ev.EventID.String(),
				AppId:        ev.Source,
				Body:         body,
			},
		)
		cancel()

		if err != nil {
			p.logger.Error("failed publishing outbox event to rabbitmq", "event_id", ev.EventID, "error", err)
			_ = p.repo.MarkOutboxEventFailed(ctx, ev.ID, err.Error())
			var remainingIDs []int64
			for _, rem := range events[i+1:] {
				remainingIDs = append(remainingIDs, rem.ID)
			}
			if len(remainingIDs) > 0 {
				if rErr := p.repo.ResetOutboxEventsToPending(ctx, remainingIDs); rErr != nil {
					p.logger.Error("failed resetting unprocessed outbox events to pending", "error", rErr)
				}
			}
			// broker might be broken; force reconnect on next cycle
			p.closeBroker()
			return
		}

		if err := p.repo.MarkOutboxEventPublished(ctx, ev.ID); err != nil {
			p.logger.Error("failed marking outbox event published", "event_id", ev.EventID, "error", err)
		}
	}
}
