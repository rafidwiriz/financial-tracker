CREATE TABLE accounts (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID NOT NULL REFERENCES users(id),
    name        VARCHAR(100) NOT NULL,
    type        VARCHAR(20) NOT NULL CONSTRAINT accounts_type_check CHECK (type IN ('checking','savings','credit_card','cash','investment')),
    balance     BIGINT NOT NULL DEFAULT 0,
    currency    VARCHAR(3) NOT NULL,
    is_closed   BOOLEAN NOT NULL DEFAULT false,
    color       VARCHAR(7),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX accounts_user_id_idx ON accounts (user_id);
