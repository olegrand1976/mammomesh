-- MammoMesh : schéma initial du socle (suivi, comptes, temps réel, MCP).
-- Les tables de la banque d'entraînement arriveront avec la chaîne d'import (jalon 0-1).

CREATE TABLE users (
    id              BIGSERIAL PRIMARY KEY,
    username        TEXT NOT NULL UNIQUE,
    display_name    TEXT NOT NULL DEFAULT '',
    password_hash   TEXT NOT NULL,
    role            TEXT NOT NULL CHECK (role IN ('admin', 'responsable_scientifique', 'data_manager', 'radiologue', 'lecteur')),
    -- Prévu pour le double facteur, vide tant qu'il n'est pas activé.
    totp_secret_enc BYTEA,
    active          BOOLEAN NOT NULL DEFAULT TRUE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE sessions (
    id           BIGSERIAL PRIMARY KEY,
    token_hash   BYTEA NOT NULL UNIQUE,
    user_id      BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    last_seen_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    expires_at   TIMESTAMPTZ NOT NULL
);
CREATE INDEX sessions_user_idx ON sessions (user_id);

CREATE TABLE mcp_tokens (
    id           BIGSERIAL PRIMARY KEY,
    user_id      BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name         TEXT NOT NULL,
    token_hash   BYTEA NOT NULL UNIQUE,
    scope        TEXT NOT NULL CHECK (scope IN ('lecture', 'proposition')),
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    expires_at   TIMESTAMPTZ NOT NULL,
    last_used_at TIMESTAMPTZ,
    revoked_at   TIMESTAMPTZ
);
CREATE INDEX mcp_tokens_user_idx ON mcp_tokens (user_id);

CREATE TABLE campaigns (
    id              BIGSERIAL PRIMARY KEY,
    name            TEXT NOT NULL,
    lineage         TEXT NOT NULL,
    config          JSONB NOT NULL,
    config_hash     TEXT NOT NULL,
    state           TEXT NOT NULL CHECK (state IN ('en_cours', 'en_pause', 'arretee', 'terminee', 'en_echec')),
    generation      INT NOT NULL DEFAULT 0,
    max_generations INT NOT NULL,
    budget_total    DOUBLE PRECISION NOT NULL,
    budget_used     DOUBLE PRECISION NOT NULL DEFAULT 0,
    best_sensitivity DOUBLE PRECISION,
    stagnation      INT NOT NULL DEFAULT 0,
    created_by      BIGINT REFERENCES users(id),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    started_at      TIMESTAMPTZ,
    finished_at     TIMESTAMPTZ,
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX campaigns_state_idx ON campaigns (state);

CREATE TABLE config_drafts (
    id                   BIGSERIAL PRIMARY KEY,
    base_campaign_id     BIGINT REFERENCES campaigns(id),
    config               JSONB NOT NULL,
    diff                 JSONB NOT NULL DEFAULT '[]',
    author_id            BIGINT NOT NULL REFERENCES users(id),
    source               TEXT NOT NULL CHECK (source IN ('interface', 'mcp')),
    checks               JSONB,
    verified_ok          BOOLEAN NOT NULL DEFAULT FALSE,
    verified_at          TIMESTAMPTZ,
    launched_campaign_id BIGINT REFERENCES campaigns(id),
    created_at           TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Agrégats par génération : l'écran de suivi et le MCP les lisent sans rien recalculer.
CREATE TABLE generations (
    campaign_id        BIGINT NOT NULL REFERENCES campaigns(id) ON DELETE CASCADE,
    number             INT NOT NULL,
    kind               TEXT NOT NULL DEFAULT 'rapide' CHECK (kind IN ('rapide', 'lente')),
    started_at         TIMESTAMPTZ NOT NULL,
    finished_at        TIMESTAMPTZ NOT NULL,
    population         INT NOT NULL,
    feasible           INT NOT NULL,
    -- NULL quand aucun individu de la génération n'est réalisable.
    best_sensitivity   DOUBLE PRECISION,
    median_sensitivity DOUBLE PRECISION NOT NULL,
    best_cost_ms       DOUBLE PRECISION NOT NULL,
    median_cost_ms     DOUBLE PRECISION NOT NULL,
    eval_gap           DOUBLE PRECISION,
    PRIMARY KEY (campaign_id, number)
);

CREATE TABLE individuals (
    id               BIGSERIAL PRIMARY KEY,
    campaign_id      BIGINT NOT NULL REFERENCES campaigns(id) ON DELETE CASCADE,
    generation       INT NOT NULL,
    name             TEXT NOT NULL,
    parent_name      TEXT,
    mutations        TEXT[] NOT NULL DEFAULT '{}',
    genome           JSONB NOT NULL,
    sensitivity      DOUBLE PRECISION NOT NULL,
    sensitivity_low  DOUBLE PRECISION NOT NULL,
    sensitivity_high DOUBLE PRECISION NOT NULL,
    sensitivity_eval DOUBLE PRECISION NOT NULL,
    cost_ms          DOUBLE PRECISION NOT NULL,
    feasible         BOOLEAN NOT NULL,
    on_front         BOOLEAN NOT NULL DEFAULT FALSE
);
CREATE INDEX individuals_front_idx ON individuals (campaign_id, generation) WHERE on_front;
CREATE UNIQUE INDEX individuals_name_idx ON individuals (campaign_id, name);

-- Journal : source du récit textuel et du flux temps réel (id = identifiant SSE).
CREATE TABLE events (
    id          BIGSERIAL PRIMARY KEY,
    campaign_id BIGINT REFERENCES campaigns(id) ON DELETE CASCADE,
    type        TEXT NOT NULL,
    message     TEXT NOT NULL,
    payload     JSONB NOT NULL DEFAULT '{}',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX events_campaign_idx ON events (campaign_id, id);

-- Avis temps réel : envoyé par PostgreSQL au commit seulement, message minimal (< 8 000 octets).
CREATE FUNCTION notify_event() RETURNS trigger AS $$
BEGIN
    PERFORM pg_notify('mammomesh_events',
        json_build_object('id', NEW.id, 'campaign_id', NEW.campaign_id, 'type', NEW.type)::text);
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER events_notify AFTER INSERT ON events
    FOR EACH ROW EXECUTE FUNCTION notify_event();

CREATE TABLE audit_log (
    id         BIGSERIAL PRIMARY KEY,
    user_id    BIGINT REFERENCES users(id),
    source     TEXT NOT NULL CHECK (source IN ('interface', 'mcp', 'systeme')),
    action     TEXT NOT NULL,
    details    JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX audit_log_created_idx ON audit_log (created_at);
