CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "btree_gist";

-- 1. LOAN REQUESTS
CREATE TABLE IF NOT EXISTS loan_requests (
    id UUID PRIMARY KEY,
    organization_id UUID NOT NULL,
    requester_user_id UUID NOT NULL,
    item_id UUID NOT NULL,
    quantity INT NOT NULL CHECK (quantity > 0),
    start_time TIMESTAMPTZ NOT NULL,
    end_time TIMESTAMPTZ NOT NULL,
    purpose TEXT NOT NULL,
    notes TEXT,
    status VARCHAR(32) NOT NULL DEFAULT 'pending'
        CHECK (status IN ('pending', 'approved', 'rejected', 'cancelled')),
    reviewer_user_id UUID,
    review_notes TEXT,
    reviewed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT check_loan_request_times CHECK (start_time < end_time)
);
CREATE INDEX IF NOT EXISTS idx_loan_requests_org_status ON loan_requests (organization_id, status);
CREATE INDEX IF NOT EXISTS idx_loan_requests_requester ON loan_requests (requester_user_id);
CREATE INDEX IF NOT EXISTS idx_loan_requests_item_id ON loan_requests (item_id);

-- 2. RESERVATIONS
CREATE TABLE IF NOT EXISTS reservations (
    id UUID PRIMARY KEY,
    organization_id UUID NOT NULL,
    loan_request_id UUID NOT NULL REFERENCES loan_requests(id) ON DELETE CASCADE,
    unit_id UUID NOT NULL,
    start_time TIMESTAMPTZ NOT NULL,
    end_time TIMESTAMPTZ NOT NULL,
    status VARCHAR(32) NOT NULL DEFAULT 'active'
        CHECK (status IN ('active', 'fulfilled', 'cancelled')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT check_reservation_times CHECK (start_time < end_time),
    CONSTRAINT exclude_overlapping_reservations EXCLUDE USING gist (
        unit_id WITH =,
        tstzrange(start_time, end_time, '[)') WITH &&
    ) WHERE (status = 'active')
);
CREATE INDEX IF NOT EXISTS idx_reservations_unit_status ON reservations (unit_id, status);
CREATE INDEX IF NOT EXISTS idx_reservations_request ON reservations (loan_request_id);

-- 3. LOANS
CREATE TABLE IF NOT EXISTS loans (
    id UUID PRIMARY KEY,
    organization_id UUID NOT NULL,
    loan_request_id UUID NOT NULL REFERENCES loan_requests(id) ON DELETE RESTRICT,
    reservation_id UUID NOT NULL REFERENCES reservations(id) ON DELETE RESTRICT,
    unit_id UUID NOT NULL,
    borrower_user_id UUID NOT NULL,
    status VARCHAR(32) NOT NULL DEFAULT 'pending_checkout'
        CHECK (status IN ('pending_checkout', 'checked_out', 'returned', 'overdue', 'cancelled')),
    checkout_time TIMESTAMPTZ,
    expected_return_time TIMESTAMPTZ NOT NULL,
    actual_return_time TIMESTAMPTZ,
    return_condition VARCHAR(32)
        CHECK (return_condition IS NULL OR return_condition IN ('new', 'good', 'fair', 'poor', 'damaged')),
    return_notes TEXT,
    processed_by_user_id UUID,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_loans_org_status ON loans (organization_id, status);
CREATE INDEX IF NOT EXISTS idx_loans_borrower ON loans (borrower_user_id);
CREATE INDEX IF NOT EXISTS idx_loans_unit_id ON loans (unit_id);
CREATE INDEX IF NOT EXISTS idx_loans_overdue_scan ON loans (status, expected_return_time)
    WHERE status = 'checked_out';

-- 4. CHECKOUT TOKENS
CREATE TABLE IF NOT EXISTS checkout_tokens (
    id UUID PRIMARY KEY,
    loan_id UUID NOT NULL REFERENCES loans(id) ON DELETE CASCADE,
    token_hash VARCHAR(64) NOT NULL UNIQUE,
    expires_at TIMESTAMPTZ NOT NULL,
    used_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_checkout_tokens_lookup ON checkout_tokens (token_hash)
    WHERE used_at IS NULL;

-- 5. STATUS HISTORY
CREATE TABLE IF NOT EXISTS status_history (
    id UUID PRIMARY KEY,
    entity_type VARCHAR(32) NOT NULL CHECK (entity_type IN ('loan_request', 'loan', 'reservation')),
    entity_id UUID NOT NULL,
    from_status VARCHAR(32) NOT NULL,
    to_status VARCHAR(32) NOT NULL,
    changed_by_user_id UUID NOT NULL,
    reason TEXT,
    correlation_id VARCHAR(255),
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_status_history_entity ON status_history (entity_type, entity_id, created_at);

-- 6. TRANSACTIONAL OUTBOX
CREATE TABLE IF NOT EXISTS outbox_events (
    id BIGSERIAL PRIMARY KEY,
    event_id UUID NOT NULL UNIQUE,
    event_type VARCHAR(100) NOT NULL,
    event_version INT NOT NULL DEFAULT 1,
    source VARCHAR(100) NOT NULL DEFAULT 'lending-service',
    subject VARCHAR(255) NOT NULL,
    organization_id UUID NOT NULL,
    occurred_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    correlation_id VARCHAR(255) NOT NULL,
    causation_id VARCHAR(255) NOT NULL,
    data JSONB NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'in_flight', 'published', 'failed')),
    retry_count INT NOT NULL DEFAULT 0,
    last_error TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    published_at TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS idx_outbox_events_pending ON outbox_events (status, id ASC)
    WHERE status = 'pending';

-- 7. IDEMPOTENCY KEYS
CREATE TABLE IF NOT EXISTS idempotency_keys (
    key VARCHAR(255) NOT NULL,
    organization_id UUID NOT NULL,
    user_id UUID NOT NULL,
    request_path VARCHAR(255) NOT NULL,
    request_hash VARCHAR(64) NOT NULL,
    response_code INT NOT NULL,
    response_body JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMPTZ NOT NULL,
    PRIMARY KEY (organization_id, key)
);
CREATE INDEX IF NOT EXISTS idx_idempotency_expires_at ON idempotency_keys (expires_at);
