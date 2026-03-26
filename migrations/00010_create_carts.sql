-- +goose Up
-- +goose StatementBegin

CREATE TYPE cart_status AS ENUM ('active', 'abandoned', 'converted');

CREATE TABLE carts (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID REFERENCES users(id) ON DELETE CASCADE,
    session_id  TEXT UNIQUE,
    status      cart_status NOT NULL DEFAULT 'active',
    expires_at  TIMESTAMPTZ NOT NULL DEFAULT now() + INTERVAL '7 days',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_carts_user_id ON carts(user_id);
CREATE INDEX idx_carts_session_id ON carts(session_id);
CREATE INDEX idx_carts_expires_at ON carts(expires_at);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP TABLE IF EXISTS carts;
DROP TYPE IF EXISTS cart_status;

-- +goose StatementEnd