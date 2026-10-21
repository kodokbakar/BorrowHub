\set ON_ERROR_STOP on

-- ============================================================================
-- 1. IDENTITY SERVICE SEED
-- ============================================================================
\connect borrowhub_identity

CREATE SCHEMA IF NOT EXISTS development;

CREATE TABLE IF NOT EXISTS development.seed (
    key text PRIMARY KEY,
    value jsonb NOT NULL
);

INSERT INTO development.seed (key, value)
VALUES ('bootstrap', '{"service":"identity","version":"1.0.0"}')
ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value;

-- Users
-- Password for all: BorrowHub123!
-- Argon2id hash: $argon2id$v=19$m=65536,t=3,p=2$T1CUYhMh/jidkZbvfbBeZg$qYUPtLdSadsgQVRkB5mvO1g7RWJ7xC6Se7on45u/sMI
INSERT INTO users (id, email, password_hash, display_name, status, terms_accepted_at, created_at, updated_at)
VALUES
    ('10000000-0000-0000-0000-000000000001', 'admin@borrowhub.local', '$argon2id$v=19$m=65536,t=3,p=2$T1CUYhMh/jidkZbvfbBeZg$qYUPtLdSadsgQVRkB5mvO1g7RWJ7xC6Se7on45u/sMI', 'Admin Acme', 'active', '2026-01-01 00:00:00+00', '2026-01-01 00:00:00+00', '2026-01-01 00:00:00+00'),
    ('10000000-0000-0000-0000-000000000002', 'member@borrowhub.local', '$argon2id$v=19$m=65536,t=3,p=2$T1CUYhMh/jidkZbvfbBeZg$qYUPtLdSadsgQVRkB5mvO1g7RWJ7xC6Se7on45u/sMI', 'Member Acme', 'active', '2026-01-01 00:00:00+00', '2026-01-01 00:00:00+00', '2026-01-01 00:00:00+00'),
    ('10000000-0000-0000-0000-000000000003', 'researcher@borrowhub.local', '$argon2id$v=19$m=65536,t=3,p=2$T1CUYhMh/jidkZbvfbBeZg$qYUPtLdSadsgQVRkB5mvO1g7RWJ7xC6Se7on45u/sMI', 'Researcher Lab', 'active', '2026-01-01 00:00:00+00', '2026-01-01 00:00:00+00', '2026-01-01 00:00:00+00')
ON CONFLICT (id) DO NOTHING;

-- Organizations
INSERT INTO organizations (id, name, slug, timezone, visibility, max_loan_duration_days, status, created_at, updated_at)
VALUES
    ('a0000000-0000-0000-0000-000000000001', 'Acme Makerspace', 'acme-makerspace', 'Asia/Jakarta', 'public', 14, 'active', '2026-01-01 00:00:00+00', '2026-01-01 00:00:00+00'),
    ('a0000000-0000-0000-0000-000000000002', 'Lab Riset', 'lab-riset', 'Asia/Jakarta', 'private', 30, 'active', '2026-01-01 00:00:00+00', '2026-01-01 00:00:00+00')
ON CONFLICT (id) DO NOTHING;

-- Memberships
INSERT INTO memberships (id, organization_id, user_id, role, created_at, updated_at)
VALUES
    ('20000000-0000-0000-0000-000000000001', 'a0000000-0000-0000-0000-000000000001', '10000000-0000-0000-0000-000000000001', 'owner', '2026-01-01 00:00:00+00', '2026-01-01 00:00:00+00'),
    ('20000000-0000-0000-0000-000000000002', 'a0000000-0000-0000-0000-000000000001', '10000000-0000-0000-0000-000000000002', 'member', '2026-01-01 00:00:00+00', '2026-01-01 00:00:00+00'),
    ('20000000-0000-0000-0000-000000000003', 'a0000000-0000-0000-0000-000000000002', '10000000-0000-0000-0000-000000000001', 'admin', '2026-01-01 00:00:00+00', '2026-01-01 00:00:00+00'),
    ('20000000-0000-0000-0000-000000000004', 'a0000000-0000-0000-0000-000000000002', '10000000-0000-0000-0000-000000000003', 'member', '2026-01-01 00:00:00+00', '2026-01-01 00:00:00+00')
ON CONFLICT (id) DO NOTHING;

-- Sample invitation
INSERT INTO invitations (id, organization_id, email, role, token_hash, invited_by_user_id, status, expires_at, created_at, updated_at)
VALUES
    ('30000000-0000-0000-0000-000000000001', 'a0000000-0000-0000-0000-000000000001', 'newbie@borrowhub.local', 'member', 'e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855', '10000000-0000-0000-0000-000000000001', 'pending', '2027-01-01 00:00:00+00', '2026-01-01 00:00:00+00', '2026-01-01 00:00:00+00')
ON CONFLICT (id) DO NOTHING;


-- ============================================================================
-- 2. INVENTORY SERVICE SEED
-- ============================================================================
\connect borrowhub_inventory

CREATE SCHEMA IF NOT EXISTS development;

CREATE TABLE IF NOT EXISTS development.seed (
    key text PRIMARY KEY,
    value jsonb NOT NULL
);

INSERT INTO development.seed (key, value)
VALUES ('bootstrap', '{"service":"inventory","version":"1.0.0"}')
ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value;

-- Categories
INSERT INTO categories (id, organization_id, name, slug, description, created_at, updated_at)
VALUES
    ('c0000000-0000-0000-0000-000000000001', 'a0000000-0000-0000-0000-000000000001', 'Kamera', 'kamera', 'Peralatan fotografi dan videografi profesional', '2026-01-01 00:00:00+00', '2026-01-01 00:00:00+00'),
    ('c0000000-0000-0000-0000-000000000002', 'a0000000-0000-0000-0000-000000000001', 'Laptop', 'laptop', 'Laptop untuk komputasi, pengembangan, dan editing', '2026-01-01 00:00:00+00', '2026-01-01 00:00:00+00'),
    ('c0000000-0000-0000-0000-000000000003', 'a0000000-0000-0000-0000-000000000001', 'Audio', 'audio', 'Mikrofon, mixer, dan periferal tata suara', '2026-01-01 00:00:00+00', '2026-01-01 00:00:00+00'),
    ('c0000000-0000-0000-0000-000000000004', 'a0000000-0000-0000-0000-000000000002', 'Instrumen Lab', 'instrumen-lab', 'Alat ukur dan eksperimen riset laboratorium', '2026-01-01 00:00:00+00', '2026-01-01 00:00:00+00')
ON CONFLICT (id) DO NOTHING;

-- Items
INSERT INTO items (id, organization_id, category_id, name, description, is_lendable, created_at, updated_at)
VALUES
    ('d0000000-0000-0000-0000-000000000001', 'a0000000-0000-0000-0000-000000000001', 'c0000000-0000-0000-0000-000000000001', 'Sony Cinema Line FX3', 'Full-frame cinema camera dengan resolusi 4K 120fps', true, '2026-01-01 00:00:00+00', '2026-01-01 00:00:00+00'),
    ('d0000000-0000-0000-0000-000000000002', 'a0000000-0000-0000-0000-000000000001', 'c0000000-0000-0000-0000-000000000002', 'Apple MacBook Pro 14 M3', '14-inch MacBook Pro M3 Pro 18GB Unified Memory 512GB SSD', true, '2026-01-01 00:00:00+00', '2026-01-01 00:00:00+00'),
    ('d0000000-0000-0000-0000-000000000003', 'a0000000-0000-0000-0000-000000000001', 'c0000000-0000-0000-0000-000000000003', 'Rode Wireless GO II', 'Dual-channel ultra-compact wireless microphone system', true, '2026-01-01 00:00:00+00', '2026-01-01 00:00:00+00'),
    ('d0000000-0000-0000-0000-000000000004', 'a0000000-0000-0000-0000-000000000002', 'c0000000-0000-0000-0000-000000000004', 'Rigol DS1054Z Digital Oscilloscope', '4-channel 50MHz digital oscilloscope dengan bandwidth upgradeable', true, '2026-01-01 00:00:00+00', '2026-01-01 00:00:00+00')
ON CONFLICT (id) DO NOTHING;

-- Units
INSERT INTO units (id, organization_id, item_id, barcode, serial_number, status, condition, notes, created_at, updated_at)
VALUES
    ('e0000000-0000-0000-0000-000000000001', 'a0000000-0000-0000-0000-000000000001', 'd0000000-0000-0000-0000-000000000001', 'ACME-CAM-001', 'FX3-SN-882101', 'available', 'good', 'Lensa kit 24-70mm terpasang, baterai ekstra 2x', '2026-01-01 00:00:00+00', '2026-01-01 00:00:00+00'),
    ('e0000000-0000-0000-0000-000000000002', 'a0000000-0000-0000-0000-000000000001', 'd0000000-0000-0000-0000-000000000002', 'ACME-LAP-001', 'MBP-SN-994202', 'borrowed', 'new', 'Sudah terinstall Docker, VS Code, dan toolchains', '2026-01-01 00:00:00+00', '2026-01-01 00:00:00+00'),
    ('e0000000-0000-0000-0000-000000000003', 'a0000000-0000-0000-0000-000000000001', 'd0000000-0000-0000-0000-000000000003', 'ACME-PRN-001', 'RDE-SN-113303', 'available', 'fair', 'Kabel jack 3.5mm dan case lengkap', '2026-01-01 00:00:00+00', '2026-01-01 00:00:00+00')
ON CONFLICT (id) DO NOTHING;

-- Condition logs
INSERT INTO condition_logs (id, unit_id, previous_condition, new_condition, reason, changed_by, created_at)
VALUES
    ('f0000000-0000-0000-0000-000000000001', 'e0000000-0000-0000-0000-000000000001', 'new', 'good', 'Inspeksi awal inventarisasi tahunan', '10000000-0000-0000-0000-000000000001', '2026-01-01 00:00:00+00')
ON CONFLICT (id) DO NOTHING;


-- ============================================================================
-- 3. LENDING SERVICE SEED
-- ============================================================================
\connect borrowhub_lending

CREATE SCHEMA IF NOT EXISTS development;

CREATE TABLE IF NOT EXISTS development.seed (
    key text PRIMARY KEY,
    value jsonb NOT NULL
);

INSERT INTO development.seed (key, value)
VALUES ('bootstrap', '{"service":"lending","version":"1.0.0"}')
ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value;

-- Loan Requests
INSERT INTO loan_requests (id, organization_id, requester_user_id, item_id, quantity, start_time, end_time, purpose, notes, status, reviewer_user_id, review_notes, reviewed_at, created_at, updated_at)
VALUES
    ('70000000-0000-0000-0000-000000000001', 'a0000000-0000-0000-0000-000000000001', '10000000-0000-0000-0000-000000000002', 'd0000000-0000-0000-0000-000000000002', 1, '2026-10-20 09:00:00+00', '2026-10-28 18:00:00+00', 'Penyusunan modul pelatihan IoT dan embedded system', 'Memerlukan laptop dengan performa tinggi', 'approved', '10000000-0000-0000-0000-000000000001', 'Disetujui untuk kegiatan lab', '2026-10-20 09:30:00+00', '2026-10-20 09:15:00+00', '2026-10-20 09:30:00+00')
ON CONFLICT (id) DO NOTHING;

-- Reservations
INSERT INTO reservations (id, organization_id, loan_request_id, unit_id, start_time, end_time, status, created_at, updated_at)
VALUES
    ('71000000-0000-0000-0000-000000000001', 'a0000000-0000-0000-0000-000000000001', '70000000-0000-0000-0000-000000000001', 'e0000000-0000-0000-0000-000000000002', '2026-10-20 09:00:00+00', '2026-10-28 18:00:00+00', 'fulfilled', '2026-10-20 09:30:00+00', '2026-10-20 10:00:00+00')
ON CONFLICT (id) DO NOTHING;

-- Loans
INSERT INTO loans (id, organization_id, loan_request_id, reservation_id, unit_id, borrower_user_id, status, checkout_time, expected_return_time, actual_return_time, return_condition, return_notes, processed_by_user_id, created_at, updated_at)
VALUES
    ('72000000-0000-0000-0000-000000000001', 'a0000000-0000-0000-0000-000000000001', '70000000-0000-0000-0000-000000000001', '71000000-0000-0000-0000-000000000001', 'e0000000-0000-0000-0000-000000000002', '10000000-0000-0000-0000-000000000002', 'checked_out', '2026-10-20 10:00:00+00', '2026-10-28 18:00:00+00', NULL, NULL, NULL, '10000000-0000-0000-0000-000000000001', '2026-10-20 09:30:00+00', '2026-10-20 10:00:00+00')
ON CONFLICT (id) DO NOTHING;

-- Checkout Tokens
-- token hash for dummy token 'token-seed-001'
INSERT INTO checkout_tokens (id, loan_id, token_hash, expires_at, used_at, created_at)
VALUES
    ('73000000-0000-0000-0000-000000000001', '72000000-0000-0000-0000-000000000001', 'b2b8d00346fb8e5e8e815e1bc3a85ff6705ec6866ec8f6a9170e176df91ee738', '2026-10-21 09:30:00+00', '2026-10-20 10:00:00+00', '2026-10-20 09:30:00+00')
ON CONFLICT (id) DO NOTHING;

-- Status History
INSERT INTO status_history (id, entity_type, entity_id, from_status, to_status, changed_by_user_id, reason, correlation_id, created_at)
VALUES
    ('74000000-0000-0000-0000-000000000001', 'loan_request', '70000000-0000-0000-0000-000000000001', 'pending', 'approved', '10000000-0000-0000-0000-000000000001', 'Disetujui admin', 'seed-corr-001', '2026-10-20 09:30:00+00'),
    ('74000000-0000-0000-0000-000000000002', 'loan', '72000000-0000-0000-0000-000000000001', 'pending_checkout', 'checked_out', '10000000-0000-0000-0000-000000000001', 'Token checkout divalidasi', 'seed-corr-002', '2026-10-20 10:00:00+00')
ON CONFLICT (id) DO NOTHING;

-- Outbox Events
INSERT INTO outbox_events (event_id, event_type, event_version, source, subject, organization_id, occurred_at, correlation_id, causation_id, data, status, retry_count, published_at)
VALUES
    ('75000000-0000-0000-0000-000000000001', 'loan.approved', 1, 'lending-service', 'loans/72000000-0000-0000-0000-000000000001', 'a0000000-0000-0000-0000-000000000001', '2026-10-20 09:30:00+00', 'seed-corr-001', '70000000-0000-0000-0000-000000000001', '{"loan_id": "72000000-0000-0000-0000-000000000001", "borrower_user_id": "10000000-0000-0000-0000-000000000002"}'::jsonb, 'published', 0, '2026-10-20 09:30:01+00'),
    ('75000000-0000-0000-0000-000000000002', 'loan.checked_out', 1, 'lending-service', 'loans/72000000-0000-0000-0000-000000000001', 'a0000000-0000-0000-0000-000000000001', '2026-10-20 10:00:00+00', 'seed-corr-002', '72000000-0000-0000-0000-000000000001', '{"loan_id": "72000000-0000-0000-0000-000000000001", "unit_id": "e0000000-0000-0000-0000-000000000002"}'::jsonb, 'published', 0, '2026-10-20 10:00:01+00')
ON CONFLICT (event_id) DO NOTHING;


-- ============================================================================
-- 4. NOTIFICATION SERVICE SEED
-- ============================================================================
\connect borrowhub_notification

CREATE SCHEMA IF NOT EXISTS development;

CREATE TABLE IF NOT EXISTS development.seed (
    key text PRIMARY KEY,
    value jsonb NOT NULL
);

INSERT INTO development.seed (key, value)
VALUES ('bootstrap', '{"service":"notification","version":"1.0.0"}')
ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value;

-- Processed events
INSERT INTO processed_events (event_id, event_type, processed_at, status)
VALUES
    ('75000000-0000-0000-0000-000000000001', 'loan.approved', '2026-10-20 09:30:05+00', 'success'),
    ('75000000-0000-0000-0000-000000000002', 'loan.checked_out', '2026-10-20 10:00:05+00', 'success')
ON CONFLICT (event_id) DO NOTHING;

-- Notifications
INSERT INTO notifications (id, organization_id, recipient_user_id, event_id, event_type, title, message, data, status, created_at, updated_at)
VALUES
    ('80000000-0000-0000-0000-000000000001', 'a0000000-0000-0000-0000-000000000001', '10000000-0000-0000-0000-000000000002', '75000000-0000-0000-0000-000000000001', 'loan.approved', 'Permintaan Peminjaman Disetujui', 'Permintaan peminjaman Apple MacBook Pro 14 M3 Anda telah disetujui.', '{"loan_id": "72000000-0000-0000-0000-000000000001"}'::jsonb, 'unread', '2026-10-20 09:30:05+00', '2026-10-20 09:30:05+00'),
    ('80000000-0000-0000-0000-000000000002', 'a0000000-0000-0000-0000-000000000001', '10000000-0000-0000-0000-000000000002', '75000000-0000-0000-0000-000000000002', 'loan.checked_out', 'Peralatan Berhasil Di-checkout', 'Unit MacBook Pro 14 M3 telah berhasil diambil.', '{"loan_id": "72000000-0000-0000-0000-000000000001"}'::jsonb, 'read', '2026-10-20 10:00:05+00', '2026-10-20 10:05:00+00')
ON CONFLICT (id) DO NOTHING;

-- Deliveries
INSERT INTO deliveries (id, notification_id, channel, recipient, status, attempts_count, created_at, updated_at)
VALUES
    ('81000000-0000-0000-0000-000000000001', '80000000-0000-0000-0000-000000000001', 'in_app', '10000000-0000-0000-0000-000000000002', 'delivered', 1, '2026-10-20 09:30:05+00', '2026-10-20 09:30:05+00'),
    ('81000000-0000-0000-0000-000000000002', '80000000-0000-0000-0000-000000000001', 'email', 'member@borrowhub.local', 'delivered', 1, '2026-10-20 09:30:06+00', '2026-10-20 09:30:06+00'),
    ('81000000-0000-0000-0000-000000000003', '80000000-0000-0000-0000-000000000002', 'in_app', '10000000-0000-0000-0000-000000000002', 'delivered', 1, '2026-10-20 10:00:05+00', '2026-10-20 10:00:05+00')
ON CONFLICT (id) DO NOTHING;
