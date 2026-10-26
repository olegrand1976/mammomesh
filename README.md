# MammoMesh

Recherche autonome d'un couple **CNN + CephaloMesh** pour repérer les signes de cancer du sein sur mammographie, par algorithme génétique et recherche d'architecture évolutive. Le CNN voit les lésions possibles ; CephaloMesh raisonne sur ce qui a été vu (concordance des vues, symétrie, évolution depuis l'examen précédent, cohérence BI-RADS).

> **Outil de recherche.** Aucun résultat ne sert à un diagnostic. Un usage clinique exigerait le marquage CE (règlement (UE) 2017/745) et la démarche décrite dans la note de conception.

La conception complète (protocole, garde-fous, pile, déploiement) est dans la note de conception MammoMesh. Ce dépôt en est la mise en œuvre.

## État : socle exécutable

| Brique | État |
|---|---|
| API Go : login, campagnes, brouillons de configuration, contrôles avant lancement, journal d'audit | fait |
| Temps réel : PostgreSQL `LISTEN/NOTIFY` → relais Go → SSE, fusion des progressions, reprise par `Last-Event-ID` | fait |
| Orchestrateur autonome : verrou unique, reprise après redémarrage, arrêt sur plafond, budget ou stagnation de E-val | fait |
| Serveur MCP pour Claude Code (`/mcp`) : 10 outils, ressource « rapport », 2 invites, jetons personnels | fait |
| Interface Nuxt : page de présentation scientifique, login, campagnes, écran de suivi en direct, jetons MCP | fait |
| Docker compose : Caddy, web, api, orchestrator, migrate, PostgreSQL 18, Redis 8, SeaweedFS, sauvegarde, observabilité | fait |
| Boucle CNN + CephaloMesh | **simulée** (déterministe) : le vrai couple arrive aux jalons 1 à 3 |
| Banque d'entraînement, chaîne d'import DICOM, moteur d'analyse, Triton, entraînement | jalons 0 et 1 |

## Architecture

```
navigateur ─┐                        ┌─ web (Nuxt) : présentation pré-rendue, interface de suivi
            ├─ HTTPS ─ caddy (443) ──┤
Claude Code ┘                        └─ api (Go) : REST, SSE, login, MCP ─┐
                                                                         ├─ postgres 18 (données de suivi)
                     orchestrator (Go) : campagnes, évolution ───────────┤─ redis 8 (compteurs, cache)
                     migrate (Go) : schéma, premier administrateur ──────┘─ s3 SeaweedFS (archive, à venir)
```

Seul Caddy publie un port, sur le réseau local. Les réseaux `app`, `data` et `obs` sont internes.

## Démarrer la pile (serveur du réseau local)

Prérequis : Docker avec le plugin compose, un nom DNS local (ou une ligne dans le fichier hosts des postes) qui pointe vers le serveur.

```bash
cp .env.example .env          # adapter SITE_ADDRESS et BIND_ADDRESS
./scripts/init-secrets.sh     # génère ./secrets (jamais commité)
docker compose up -d --build
```

- Ouvrir `https://<SITE_ADDRESS>` : page de présentation ; `/login` pour l'interface de suivi.
- Premier compte : `admin`, mot de passe dans `secrets/admin_password.txt`.
- Autres comptes (pas d'inscription libre) :

```bash
echo 'un-mot-de-passe-long' | docker compose run --rm -T migrate user create --username marie --role responsable_scientifique --name "Marie D."
```

Rôles : `admin`, `responsable_scientifique`, `data_manager`, `radiologue`, `lecteur`.

Profils optionnels : `COMPOSE_PROFILES=obs` (Prometheus + Grafana sous `/obs/`), `backup` (sauvegarde restic quotidienne vers `BACKUP_DIR`).

### Certificat

Caddy signe le site avec son autorité interne. Pour que navigateurs et Claude Code lui fassent confiance :

```bash
make ca    # copie ./mammomesh-root.crt
```

Puis l'importer dans le magasin de certificats des postes ; pour Claude Code : `export NODE_EXTRA_CA_CERTS=/chemin/mammomesh-root.crt`.

## Brancher Claude Code (MCP)

1. Dans l'interface, page **Jetons MCP** : créer un jeton « lecture » ou « proposition » (90 jours, révocable).
2. Dans ce dépôt, `.mcp.json` est déjà prêt ; il suffit d'exporter :

```bash
export MAMMOMESH_URL=https://mammomesh.lan
export MAMMOMESH_MCP_TOKEN=mmk_...
claude
```

Hors du dépôt : `claude mcp add --transport http mammomesh https://mammomesh.lan/mcp --header "Authorization: Bearer mmk_..."`.

`.claude/settings.json` autorise d'office la lecture et les propositions ; `campagne_lancer` et `campagne_reglage` demandent toujours confirmation. Chaque appel est écrit dans `audit_log` (source « mcp »).

| Outil | Type |
|---|---|
| `campagnes_lister`, `campagne_etat`, `front_pareto`, `individu_fiche`, `individus_comparer`, `journal_lire` | lecture |
| `config_brouillon`, `config_verifier` | proposition (ne lance rien) |
| `campagne_lancer`, `campagne_reglage` | action, confirmée, rôle responsable scientifique ou administrateur |

Absents par construction : test caché, jeux externes, promotion, suppression, images, SQL.

## Développement

```bash
# Base et Redis locaux, puis :
export DATABASE_URL=postgres://postgres@localhost:5432/mammomesh?sslmode=disable
export MCP_TOKEN_PEPPER=dev ADMIN_USERNAME=admin ADMIN_PASSWORD=mot-de-passe-de-dev COOKIE_SECURE=false
make dev-api            # après : go run ./cmd/mammomesh migrate
make dev-orchestrator
make dev-web            # http://localhost:3000, /api relayé vers :8080
```

Serveur MCP en stdio pour le développement : `MAMMOMESH_MCP_TOKEN=mmk_... go run ./cmd/mammomesh mcp-stdio`.

## Tests

```bash
cd backend && DATABASE_URL_TEST=postgres://.../mammomesh_test go test ./...
cd web && npx nuxt typecheck && npm run build
```

Le test de bout en bout (`internal/integration`) remet la base désignée à zéro, puis vérifie : connexion et refus, brouillon → contrôles → lancement, orchestrateur jusqu'à la fin, flux SSE en direct, reprise exacte par `Last-Event-ID`, droits par rôle, outils MCP par portée de jeton, journal d'audit, révocation.

## Notes techniques

- `backend/go.mod` reste compatible Go 1.24 ; l'image Docker compile avec Go 1.26. Le SDK MCP (v1.4) passera en v1.8 avec Go ≥ 1.25.
- Les modules `golang.org/x/*` sont pris sur leurs miroirs GitHub officiels (directives `replace`) : même code, compilable là où `golang.org` est filtré.
- MinIO est remplacé par SeaweedFS (édition communautaire de MinIO archivée en 2026).
