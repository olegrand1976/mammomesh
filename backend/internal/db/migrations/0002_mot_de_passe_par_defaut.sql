-- Le premier administrateur peut naître avec le mot de passe par défaut : il doit le changer.
ALTER TABLE users ADD COLUMN must_change_password BOOLEAN NOT NULL DEFAULT FALSE;
