-- Images de test : seuls les pixels (dans le stockage) et des métadonnées techniques sont gardés.
-- Aucune donnée patient, établissement ou date d'examen n'est stockée.
CREATE TABLE images (
    id            BIGSERIAL PRIMARY KEY,
    pixel_sha256  TEXT NOT NULL UNIQUE,
    libelle       TEXT NOT NULL,
    meta          JSONB NOT NULL,
    storage_key   TEXT NOT NULL,
    taille_octets BIGINT NOT NULL,
    uploaded_by   BIGINT REFERENCES users(id),
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Tests d'un individu sur une image (résultat simulé tant que le moteur n'est pas branché).
CREATE TABLE image_tests (
    id            BIGSERIAL PRIMARY KEY,
    image_id      BIGINT NOT NULL REFERENCES images(id) ON DELETE CASCADE,
    individual_id BIGINT NOT NULL REFERENCES individuals(id),
    resultat      JSONB NOT NULL,
    created_by    BIGINT REFERENCES users(id),
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX image_tests_image ON image_tests (image_id, created_at DESC);
