CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE TABLE users (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email           VARCHAR(255) NOT NULL UNIQUE,
    password_hash   TEXT NOT NULL,
    display_name    VARCHAR(255),
    role            VARCHAR(20) NOT NULL DEFAULT 'user',
    is_active       BOOLEAN NOT NULL DEFAULT true,
    quota_bytes     BIGINT NOT NULL DEFAULT 1073741824,
    used_bytes      BIGINT NOT NULL DEFAULT 0,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE domains (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name                VARCHAR(255) NOT NULL UNIQUE,
    owner_id            UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    is_verified         BOOLEAN NOT NULL DEFAULT false,
    mx_verified         BOOLEAN NOT NULL DEFAULT false,
    spf_verified        BOOLEAN NOT NULL DEFAULT false,
    dkim_verified       BOOLEAN NOT NULL DEFAULT false,
    dmarc_verified      BOOLEAN NOT NULL DEFAULT false,
    verification_token  VARCHAR(64) NOT NULL,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_domains_name ON domains(name);
CREATE INDEX idx_domains_owner_id ON domains(owner_id);

CREATE TABLE dkim_keys (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    domain_id   UUID NOT NULL REFERENCES domains(id) ON DELETE CASCADE,
    selector    VARCHAR(63) NOT NULL,
    private_key TEXT NOT NULL,
    public_key  TEXT NOT NULL,
    is_active   BOOLEAN NOT NULL DEFAULT true,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(domain_id, selector)
);

CREATE TABLE mailboxes (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    domain_id   UUID NOT NULL REFERENCES domains(id) ON DELETE CASCADE,
    local_part  VARCHAR(64) NOT NULL,
    address     VARCHAR(255) NOT NULL UNIQUE,
    is_active   BOOLEAN NOT NULL DEFAULT true,
    is_catchall BOOLEAN NOT NULL DEFAULT false,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(domain_id, local_part)
);

CREATE INDEX idx_mailboxes_address ON mailboxes(address);
CREATE INDEX idx_mailboxes_user_id ON mailboxes(user_id);

CREATE TABLE folders (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    mailbox_id    UUID NOT NULL REFERENCES mailboxes(id) ON DELETE CASCADE,
    name          VARCHAR(255) NOT NULL,
    special_use   VARCHAR(50),
    uid_validity  INTEGER NOT NULL DEFAULT 1,
    uid_next      INTEGER NOT NULL DEFAULT 1,
    message_count INTEGER NOT NULL DEFAULT 0,
    unseen_count  INTEGER NOT NULL DEFAULT 0,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(mailbox_id, name)
);

CREATE TABLE messages (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    folder_id       UUID NOT NULL REFERENCES folders(id) ON DELETE CASCADE,
    mailbox_id      UUID NOT NULL REFERENCES mailboxes(id) ON DELETE CASCADE,
    uid             INTEGER NOT NULL,
    message_id      VARCHAR(512),
    in_reply_to     VARCHAR(512),
    subject         TEXT,
    from_address    VARCHAR(255),
    to_addresses    TEXT[],
    cc_addresses    TEXT[],
    date            TIMESTAMPTZ,
    size_bytes      INTEGER NOT NULL DEFAULT 0,
    has_attachments BOOLEAN NOT NULL DEFAULT false,
    is_seen         BOOLEAN NOT NULL DEFAULT false,
    is_answered     BOOLEAN NOT NULL DEFAULT false,
    is_flagged      BOOLEAN NOT NULL DEFAULT false,
    is_deleted      BOOLEAN NOT NULL DEFAULT false,
    is_draft        BOOLEAN NOT NULL DEFAULT false,
    storage_key     VARCHAR(512) NOT NULL,
    spam_score      FLOAT,
    is_spam         BOOLEAN NOT NULL DEFAULT false,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(folder_id, uid)
);

CREATE INDEX idx_messages_folder_id ON messages(folder_id);
CREATE INDEX idx_messages_mailbox_id ON messages(mailbox_id);
CREATE INDEX idx_messages_message_id ON messages(message_id);
CREATE INDEX idx_messages_date ON messages(date DESC);

CREATE TABLE send_queue (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    from_address  VARCHAR(255) NOT NULL,
    to_addresses  TEXT[] NOT NULL,
    storage_key   VARCHAR(512) NOT NULL,
    status        VARCHAR(20) NOT NULL DEFAULT 'pending',
    attempts      INTEGER NOT NULL DEFAULT 0,
    max_attempts  INTEGER NOT NULL DEFAULT 5,
    next_retry_at TIMESTAMPTZ,
    last_error    TEXT,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_send_queue_status ON send_queue(status, next_retry_at);

CREATE TABLE aliases (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    source_address  VARCHAR(255) NOT NULL,
    destination     VARCHAR(255) NOT NULL,
    domain_id       UUID NOT NULL REFERENCES domains(id) ON DELETE CASCADE,
    is_active       BOOLEAN NOT NULL DEFAULT true,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE api_keys (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    key_hash    VARCHAR(128) NOT NULL UNIQUE,
    key_prefix  VARCHAR(8) NOT NULL,
    name        VARCHAR(255),
    mailbox_ids UUID[] NOT NULL,
    permissions TEXT[] NOT NULL DEFAULT '{read}',
    is_active   BOOLEAN NOT NULL DEFAULT true,
    last_used_at TIMESTAMPTZ,
    expires_at  TIMESTAMPTZ,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_api_keys_key_hash ON api_keys(key_hash);
CREATE INDEX idx_api_keys_user_id ON api_keys(user_id);

CREATE TABLE webhooks (
    id        UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id   UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    url       TEXT NOT NULL,
    secret    VARCHAR(128) NOT NULL,
    events    TEXT[] NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
