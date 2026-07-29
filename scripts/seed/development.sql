\set ON_ERROR_STOP on

\connect borrowhub_identity

CREATE SCHEMA IF NOT EXISTS development;

CREATE TABLE IF NOT EXISTS development.seed (
    key text PRIMARY KEY,
    value jsonb NOT NULL
);

INSERT INTO development.seed (key, value)
VALUES (
    'bootstrap',
    '{"service":"identity"}'
)
ON CONFLICT (key) DO UPDATE
SET value = EXCLUDED.value;

\connect borrowhub_inventory

CREATE SCHEMA IF NOT EXISTS development;

CREATE TABLE IF NOT EXISTS development.seed (
    key text PRIMARY KEY,
    value jsonb NOT NULL
);

INSERT INTO development.seed (key, value)
VALUES (
    'bootstrap',
    '{"service":"inventory"}'
)
ON CONFLICT (key) DO UPDATE
SET value = EXCLUDED.value;

\connect borrowhub_lending

CREATE SCHEMA IF NOT EXISTS development;

CREATE TABLE IF NOT EXISTS development.seed (
    key text PRIMARY KEY,
    value jsonb NOT NULL
);

INSERT INTO development.seed (key, value)
VALUES (
    'bootstrap',
    '{"service":"lending"}'
)
ON CONFLICT (key) DO UPDATE
SET value = EXCLUDED.value;

\connect borrowhub_notification

CREATE SCHEMA IF NOT EXISTS development;

CREATE TABLE IF NOT EXISTS development.seed (
    key text PRIMARY KEY,
    value jsonb NOT NULL
);

INSERT INTO development.seed (key, value)
VALUES (
    'bootstrap',
    '{"service":"notification"}'
)
ON CONFLICT (key) DO UPDATE
SET value = EXCLUDED.value;