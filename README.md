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
| Campagnes chaînées : repartir du front de la génération finale (ou d'une intermédiaire) d'une campagne finie | fait |
| Images : import DICOM pseudonymisé, visionneuse (fenêtrage, zoom), test d'un individu au choix (génération finale par défaut) | fait, résultat **simulé** |
| Documentation intégrée : fonctionnement, modèle mathématique (KaTeX), guides d'écran, lexique des mutations | fait, aussi dans `docs/` |
| Docker compose : Caddy, web, api, orchestrator, migrate, PostgreSQL 18, Redis 8, SeaweedFS, sauvegarde, observabilité | fait |
| Boucle CNN + CephaloMesh | **simulée** (déterministe) : le vrai couple arrive aux jalons 1 à 3 |
| CNN de base (EfficientNet-B0, malin ou bénin sur CMMD) | code prêt (`engine/mammomesh_engine/cnn`), suivi dans l'interface (menu Entraînements) ; pas encore branché au test d'image |
| Banque d'entraînement, chaîne d'import DICOM complète (JPEG 2000, pseudonymisation de la banque), moteur d'analyse, Triton, entraînement | jalons 0 et 1 |

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

- Ouvrir `https://<SITE_ADDRESS>` (port 443 ; le port 80 redirige) : page de présentation ; `/login` pour l'interface de suivi. Sur un seul poste : `SITE_ADDRESS=localhost` puis `https://localhost`.
- Premier compte : `admin` / `admin$1`. L'interface impose de le changer à la première connexion (12 caractères au moins) ; rien d'autre n'est accessible avant. `ADMIN_PASSWORD_RANDOM=1 ./scripts/init-secrets.sh` génère un mot de passe aléatoire à la place.
- Autres comptes (pas d'inscription libre) :

```bash
echo 'un-mot-de-passe-long' | docker compose run --rm -T migrate user create --username marie --role responsable_scientifique --name "Marie D."
```

Rôles : `admin`, `responsable_scientifique`, `data_manager`, `radiologue`, `lecteur`.

Profils optionnels : `COMPOSE_PROFILES=obs` (Prometheus + Grafana sous `/obs/`), `backup` (sauvegarde restic quotidienne de PostgreSQL vers `BACKUP_DIR` ; restauration : [deploy/backup](deploy/backup/README.md)).

### Redémarrage automatique

- **Conteneurs** : tous ont `restart: unless-stopped` (sauf `migrate`) : Docker les relance s'ils s'arrêtent, et au démarrage de la machine s'ils tournaient.
- **Pile complète au démarrage** : `./scripts/installer-services.sh` installe le service utilisateur `mammomesh-pile`, qui lance `scripts/demarrer-pile.sh` (attente de Docker, profils du matériel, `docker compose up -d`) à chaque démarrage, même après un `docker compose down`. Le script indique s'il manque `sudo systemctl enable docker.service containerd.service` ou `sudo loginctl enable-linger $USER` (démarrage sans ouvrir de session). `--retirer` désinstalle.

### Hôte

- Espace disque : garder plus de 10 Gio libres. En dessous, SeaweedFS passe ses volumes en lecture seule pour protéger PostgreSQL ; au-delà de 95 % d'occupation, il l'écrit toutes les 5 s dans son journal. Les journaux des conteneurs sont bornés à 30 Mo par service.
- Redis : `sudo sysctl -w vm.overcommit_memory=1` (et la même ligne dans `/etc/sysctl.d/99-redis.conf`), sinon la réécriture de l'AOF peut échouer.
- Messages sans conséquence au démarrage : locales absentes (PostgreSQL Alpine), `certutil` absent (Caddy), IAM/STS/SSE non configurés (SeaweedFS : le stockage S3 ne sert pas encore ; ses accès seront protégés quand les images arriveront, au jalon 1). SeaweedFS et Redis restent sur un réseau interne sans accès extérieur.

### Certificat

Caddy signe le site avec son autorité interne. Pour que navigateurs et Claude Code lui fassent confiance :

```bash
make ca    # copie ./mammomesh-root.crt
```

Puis l'importer dans le magasin de certificats des postes ; pour Claude Code : `export NODE_EXTRA_CA_CERTS=/chemin/mammomesh-root.crt`.

## Adaptation au matériel

Le même dépôt tourne sur un portable comme sur un serveur multi-GPU : rien n'est réglé à la main.

- **Hôte** : `make up` lance `scripts/detecter-materiel.sh`, qui choisit les profils compose (`nvidia`, `amd`, `intel`, `cpu`) selon les GPU présents et leur prise en charge par Docker (runtime `nvidia` de nvidia-container-toolkit, déclaré par `sudo nvidia-ctk runtime configure --runtime=docker`). `MAMMOMESH_ACCEL=cpu` force un choix.
- **Processus** : `mammomesh_engine.materiel` (dans `engine/`) détecte CPU, RAM, GPU et moteurs d'inférence, limites du conteneur comprises, et en déduit un plan :

| Réglage | Règle |
|---|---|
| Appareil | GPU dédié avec le plus de mémoire libre, puis GPU intégré, puis CPU ; un individu par GPU (jusqu'à 4 par GPU au-delà de 16 Gio) |
| Précision | BF16 si accéléré (Ampere et suivants, Arc, CPU AMX), sinon FP16 + GradScaler (Turing, Volta), sinon FP32 ; TF32 dès Ampere |
| Entrée et lot | image entière si un lot de 4 tient, sinon gradient checkpointing, sinon patchs de 512 à 256 px ; lot effectif atteint par accumulation |
| Lot mesuré | `calibrer()` fait de vrais pas d'entraînement (doublements puis dichotomie, marge de 10 %) et met le résultat en cache par appareil, modèle, taille et précision |
| Données | un cœur par entraînement, le reste en workers, bornés par la RAM libre ; `pin_memory` sur GPU |
| Inférence | TensorRT > CUDA > OpenVINO GPU > OpenVINO CPU > ONNX Runtime CPU > PyTorch CPU ; pendant un entraînement sur un GPU de moins de 16 Gio, l'inférence passe sur le GPU Intel ou le CPU |

```bash
make materiel     # profils compose, machine vue et plan
cd engine && python3 -m mammomesh_engine.materiel --json
```

`MAMMOMESH_MATERIEL=machine.json` simule une autre machine (même format que `--json`, clé `materiel`).

## Premier CNN (jalon 1, étape 1)

`mammomesh_engine.cnn` entraîne un premier CNN sur CMMD : EfficientNet-B0, qui donne la probabilité qu'un sein portant une anomalie soit malin (prouvé par biopsie). CMMD n'a pas de contours de lésions : ce modèle donne un score par image, pas de zones. Outil de recherche : aucun résultat ne sert à un diagnostic.

```bash
cd engine
python3 -m venv .venv && . .venv/bin/activate
pip install torch torchvision          # Linux : variante CUDA ; si le pilote est plus ancien, prendre la commande de pytorch.org
pip install -e '.[cnn]'
python -m mammomesh_engine.cnn --banque ../banque --limite 100 --epoques 2         # essai rapide
python -m mammomesh_engine.cnn --banque ../banque                                   # ou --banque /mnt/llit-nas/mammomesh/banque
```

- **Index** : une ligne par image étiquetée (le sein opposé, sans étiquette, est écarté) ; découpage par patiente, stratifié, en entraînement (70 %), ajustement E-ajust (15 %) et validation E-val (15 %), reproductible (`--graine`).
- **Prétraitement** : sein orienté à gauche, recadré sur le tissu, mis en 1024 × 832 (PNG 8 bits dans `./cache/cnn`, environ 2 Go). La place libre est contrôlée avant d'écrire (`MARGE_GO`, 10 Go par défaut).
- **Entraînement** : réglé par le plan matériel (appareil, précision, workers) ; sur GPU, le lot est mesuré par `calibrer()`, et la résolution descend (768 × 624, puis 512 × 416) si l'image entière ne tient pas. Poids ImageNet au départ (`--sans-preentrainement` pour partir de zéro), arrêt quand l'AUC par sein en ajustement ne progresse plus (`--patience`).
- **Résultat** dans `./modeles/cnn-cmmd-AAAAMMJJ-HHMM/` : `modele.pt` (meilleure époque, seuil inclus), `journal.csv`, `rapport.json`. Le seuil est choisi en ajustement pour une sensibilité de 0,85 par sein (`--sensibilite-cible`) ; AUC, sensibilité et spécificité finales sont mesurées en validation, sur des patientes jamais vues.
- **Reprise** : un point de reprise (`reprise.pt`, modèle, optimiseur, compteurs) est écrit à la fin de chaque époque, avec les réglages (`parametres.json`) et l'index (`index.csv`) du lancement. `python -m mammomesh_engine.cnn --reprendre modeles/cnn-cmmd-…` repart de la dernière époque finie.
- **Relance automatique** : le service `mammomesh-entrainement` (installé par `scripts/installer-services.sh`) fait tourner `python -m mammomesh_engine.cnn.auto`. Il reprend l'entraînement inachevé (machine éteinte ou en veille, processus tué, plantage) dès que le GPU est libre : aucun calcul dessus et au moins 4,5 Go de mémoire libre, deux fois à 30 s d'intervalle (`--memoire-min`). En attendant, l'interface affiche « En attente du GPU » avec la raison. Un arrêt volontaire (Ctrl+C, ou `python -m mammomesh_engine.cnn.auto --abandonner NOM`) n'est jamais repris ; après 3 reprises sans progrès, le veilleur abandonne. Journal : `modeles/NOM.log` et `journalctl --user -u mammomesh-entrainement`.
- **Relancer, supprimer** : depuis la page Entraînements (administrateur et responsable scientifique). L'API dépose une demande dans `modeles/.demandes/` (seul dossier où elle écrit, droits 777 posés par `scripts/demarrer-pile.sh` et le veilleur) ; le veilleur l'exécute dans les 5 s. « Reprendre » repart de la dernière époque finie ; « Relancer » crée un nouvel entraînement aux mêmes réglages, en file d'attente du GPU ; « Supprimer » arrête l'entraînement s'il tourne, puis efface son dossier et son journal.
- **Suivi** : menu **Entraînements** de l'interface (rafraîchi toutes les 5 s) : phase, avancement, courbes d'AUC et de perte par époque, résultat final, plan matériel. L'API lit `./modeles` en lecture seule (`MODELES_DIR`, monté d'office par Docker ; en développement : `MODELES_DIR=../modeles make dev-api`). Chaque entraînement y écrit `etat.json` (phase et pas, toutes les 10 s), `journal.csv` (une ligne par époque) et `rapport.json` (à la fin). Les fichiers doivent être lisibles par tous (umask 022 par défaut), le conteneur `api` ne tourne pas sous ton compte.
- Les poids ImageNet de torchvision viennent d'un jeu d'images sous conditions non commerciales : à réexaminer avant tout usage commercial (ou entraîner avec `--sans-preentrainement`).

## Brancher Claude Code (MCP)

1. Dans l'interface, page **Jetons MCP** : créer un jeton « lecture » ou « proposition » (90 jours, révocable).
2. Dans ce dépôt, `.mcp.json` est déjà prêt ; il suffit d'exporter :

```bash
export MAMMOMESH_URL=https://mammomesh.lan     # sur un seul poste : https://localhost
export MAMMOMESH_MCP_TOKEN=mmk_...
export NODE_EXTRA_CA_CERTS=$PWD/mammomesh-root.crt   # certificat de Caddy, voir « Certificat » (make ca)
claude
```

Hors du dépôt : `claude mcp add --transport http mammomesh https://mammomesh.lan/mcp --header "Authorization: Bearer mmk_..."`.

`.claude/settings.json` autorise d'office la lecture et les propositions ; `campagne_lancer` et `campagne_reglage` demandent toujours confirmation. Chaque appel est écrit dans `audit_log` (source « mcp »).

| Outil | Type |
|---|---|
| `campagnes_lister`, `campagne_etat`, `front_pareto`, `individu_fiche`, `individus_comparer`, `journal_lire` | lecture |
| `config_brouillon` (avec `depart_campagne_id` pour repartir d'une campagne finie), `config_verifier` | proposition (ne lance rien) |
| `campagne_lancer`, `campagne_reglage` | action, confirmée, rôle responsable scientifique ou administrateur |

Absents par construction : test caché, jeux externes, promotion, suppression, images, SQL.

## Développement

Prérequis : Go 1.27, Node 24 LTS, PostgreSQL 18 ; Redis 8 facultatif.

```bash
export DATABASE_URL=postgres://postgres@localhost:5432/mammomesh?sslmode=disable
export REDIS_URL=redis://localhost:6379/0   # ou REDIS_URL= (vide) : compteurs en mémoire, sans Redis
export MCP_TOKEN_PEPPER=dev COOKIE_SECURE=false
export IMAGES_DIR=data/images               # pixels des images de test (en Docker : IMAGES_URL, filer SeaweedFS)

cd backend && go run ./cmd/mammomesh migrate   # schéma + compte admin, une fois
make dev-api            # :8080, compte admin / admin$1 créé par migrate
make dev-orchestrator
cd web && npm ci && cd .. && make dev-web      # http://localhost:3000, /api relayé vers :8080
```

Serveur MCP en stdio pour le développement : `MAMMOMESH_MCP_TOKEN=mmk_... go run ./cmd/mammomesh mcp-stdio`.

## Tests

| Niveau | Commande | Contenu |
|---|---|---|
| Unitaires Go | `make test-backend` | mots de passe, limiteur (Redis absent, gelé, rétabli), configuration et secrets, contrôles de configuration, simulation déterministe et départ, SSE, cache MCP, import DICOM (pseudonymisation, refus), test simulé, stockages (`MAMMOMESH_FILER_TEST` : vrai filer SeaweedFS) |
| Intégration Go | `DATABASE_URL_TEST=… REDIS_URL_TEST=… make test-backend` | base remise à zéro : compte par défaut et changement imposé, origines, limitation (Redis réel), brouillon → lancement → orchestrateur, SSE et reprise `Last-Event-ID`, réglages, lectures REST, 10 outils MCP par portée, audit, révocation, repartir d'une campagne (contrôles, parents), images (import, doublons, pixels, tests, droits, purge) |
| Moteur Python | `make test-engine` | CNN de base (découpage par patiente, données cliniques, AUC et seuil, prétraitement, refus faute de place) ; plan d'exécution sur des machines types (portable A1000, serveur 2 × A100, CPU seul, Pascal, petit GPU, Arc, Iris Xe), lecture de `nvidia-smi`, machine simulée, calibrage du lot et cache |
| Unitaires web | `make test-web` | Vitest en environnement Nuxt : stores campagne et session, aides des écrans, effets calculés, lexique des mutations, formules KaTeX, concordance avec `docs/` |
| Bout en bout | `E2E_DATABASE_URL=… make e2e` | Playwright, pile complète (API, orchestrateur, Nuxt, proxy à la place de Caddy, port 14000) : présentation, refus, premier accès, campagne suivie en direct jusqu'à la fin, aides et effets des réglages, repartir d'une campagne, images (import, visionneuse, test, suppression), documentation, jetons MCP, déconnexion, affichage téléphone |

DICOM réels de sources libres (fo-dicom, pydicom, TCIA) : `make exemples` puis `go test -count=1 ./internal/imagerie -run TestExemples` ; voir [examples/README.md](examples/README.md).

La CI lance les cinq niveaux, avec PostgreSQL 18 et Redis 8 en services. Couverture Go mesurée : 80 % des instructions de `internal/`.

## Documentation

Dans l'application : menu **Documentation**. Dans le dépôt : [fonctionnement](docs/fonctionnement-campagne.md), [modèle mathématique](docs/modele-mathematique.md), guides d'écran [Nouvelle campagne](docs/guide-nouvelle-campagne.md), [suivi d'une campagne](docs/guide-suivi-campagne.md) et [Images](docs/guide-images.md), [jeux d'entraînement](docs/jeux-entrainement.md) (banque du jalon 0 : licences, téléchargement, citations, sources). Les deux versions viennent des mêmes données (`web/app/utils/`) ; un test vérifie qu'elles concordent.

## Notes techniques

- Redis ne sert qu'aux compteurs de tentatives. S'il ne répond pas, l'API passe en mémoire après un essai de 300 ms au plus, réessaie toutes les 30 s et l'écrit une fois dans le journal ; `/api/health` (champ `redis`) et la métrique `mammomesh_redis_up` en donnent l'état.
- Versions (mise à jour du 26 octobre 2026) : Go 1.27, SDK MCP v1.8 (protocole 2026-07-28), pgx 5.11, chi 5.5, go-redis 9.23 ; Nuxt 4.6, Pinia 4.1, ECharts 6.1, TypeScript 6.0 ; Node 24 LTS, Vitest 5, Playwright 1.64, KaTeX 0.19 ; DICOM : suyashkumar/dicom 1.1 ; PostgreSQL 18, Redis 8, Caddy 2.11, SeaweedFS 4.51, Prometheus 3.15, Grafana 13.2.
- TypeScript reste en 6.0 : la version 7 (réécrite en Go) n'est pas encore prise en charge par vue-tsc, qui sert au contrôle des types.
- Les modules `golang.org/x/*`, `go.uber.org/atomic` et `go.yaml.in/yaml/v3` sont pris sur leurs dépôts GitHub officiels (directives `replace`) : même code, compilable là où ces domaines sont filtrés.
- MinIO est remplacé par SeaweedFS (édition communautaire de MinIO archivée en 2026).
