CREATE TABLE IF NOT EXISTS categories (
    id UUID PRIMARY KEY,
    organization_id UUID NOT NULL,
    name TEXT NOT NULL,
    slug TEXT NOT NULL,
    description TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT uq_categories_org_slug UNIQUE (organization_id, slug)
);
CREATE INDEX IF NOT EXISTS idx_categories_org_id ON categories (organization_id);

CREATE TABLE IF NOT EXISTS items (
    id UUID PRIMARY KEY,
    organization_id UUID NOT NULL,
    category_id UUID NOT NULL REFERENCES categories(id) ON DELETE RESTRICT,
    name TEXT NOT NULL,
    description TEXT,
    is_lendable BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_items_org_id ON items (organization_id);
CREATE INDEX IF NOT EXISTS idx_items_category_id ON items (category_id);

CREATE TABLE IF NOT EXISTS units (
    id UUID PRIMARY KEY,
    organization_id UUID NOT NULL,
    item_id UUID NOT NULL REFERENCES items(id) ON DELETE RESTRICT,
    barcode TEXT NOT NULL,
    serial_number TEXT,
    status TEXT NOT NULL DEFAULT 'available' CHECK (status IN ('available', 'reserved', 'borrowed', 'maintenance', 'retired')),
    condition TEXT NOT NULL DEFAULT 'new' CHECK (condition IN ('new', 'good', 'fair', 'poor', 'damaged')),
    notes TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT uq_units_org_barcode UNIQUE (organization_id, barcode)
);
CREATE UNIQUE INDEX IF NOT EXISTS uq_units_org_serial ON units (organization_id, serial_number) WHERE serial_number IS NOT NULL AND serial_number <> '';
CREATE INDEX IF NOT EXISTS idx_units_org_id ON units (organization_id);
CREATE INDEX IF NOT EXISTS idx_units_item_id ON units (item_id);
CREATE INDEX IF NOT EXISTS idx_units_status ON units (status);

CREATE TABLE IF NOT EXISTS condition_logs (
    id UUID PRIMARY KEY,
    unit_id UUID NOT NULL REFERENCES units(id) ON DELETE CASCADE,
    previous_condition TEXT CHECK (previous_condition IS NULL OR previous_condition IN ('new', 'good', 'fair', 'poor', 'damaged')),
    new_condition TEXT NOT NULL CHECK (new_condition IN ('new', 'good', 'fair', 'poor', 'damaged')),
    reason TEXT,
    changed_by UUID NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_condition_logs_unit_id ON condition_logs (unit_id);
CREATE INDEX IF NOT EXISTS idx_condition_logs_created_at ON condition_logs (created_at);
