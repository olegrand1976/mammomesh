-- Campagnes désactivées : gardées pour l'historique, mais ignorées des listes, du MCP, des tests d'images et des départs.
ALTER TABLE campaigns ADD COLUMN disabled_at TIMESTAMPTZ;

-- Suppression d'une campagne : les brouillons perdent leur lien, les tests d'images de ses individus partent avec elle.
ALTER TABLE config_drafts DROP CONSTRAINT IF EXISTS config_drafts_base_campaign_id_fkey,
    ADD CONSTRAINT config_drafts_base_campaign_id_fkey FOREIGN KEY (base_campaign_id) REFERENCES campaigns(id) ON DELETE SET NULL;
ALTER TABLE config_drafts DROP CONSTRAINT IF EXISTS config_drafts_launched_campaign_id_fkey,
    ADD CONSTRAINT config_drafts_launched_campaign_id_fkey FOREIGN KEY (launched_campaign_id) REFERENCES campaigns(id) ON DELETE SET NULL;
ALTER TABLE image_tests DROP CONSTRAINT IF EXISTS image_tests_individual_id_fkey,
    ADD CONSTRAINT image_tests_individual_id_fkey FOREIGN KEY (individual_id) REFERENCES individuals(id) ON DELETE CASCADE;
