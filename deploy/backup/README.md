# Sauvegarde (profil `backup`)

Chaque jour à `BACKUP_HOUR` h (heure de Bruxelles, `TZ` pour une autre), `pg_dump` de la base `mammomesh` dans le dépôt restic `BACKUP_DIR/restic`, chiffré par `secrets/restic_password.txt`. Au premier démarrage, une sauvegarde est faite tout de suite. Rétention : 7 quotidiennes, 4 hebdomadaires, 12 mensuelles (`GARDER_JOURS`, `GARDER_SEMAINES`, `GARDER_MOIS`). Les images (SeaweedFS) ne sont pas comprises.

`BACKUP_DIR` doit être hors du serveur (disque externe, NAS monté sur l'hôte). Sans `secrets/restic_password.txt`, le dépôt est illisible : garder ce fichier ailleurs aussi.

Sur une machine qui n'a pas toujours accès au NAS (portable), garder `BACKUP_DIR` en local et ajouter une copie : `BACKUP_COPIE_RACINE` est le dossier de l'hôte qui contient le montage à la demande (par exemple `/mnt/llit-nas`), `BACKUP_COPIE` le dépôt vu du conteneur (`/copie/mammomesh/backup/restic`). Après chaque sauvegarde et au démarrage, les instantanés manquants y sont copiés (`restic copy`, même rétention) ; NAS injoignable, ils le seront au passage suivant. `docker compose run --rm backup copie` force la copie.

```bash
COMPOSE_PROFILES=backup docker compose up -d backup     # sauvegarde quotidienne
docker compose run --rm backup une                       # une sauvegarde tout de suite
docker compose run --rm --entrypoint restic backup snapshots
```

## Restaurer

Remplace toute la base par l'instantané : arrêter d'abord ce qui écrit dedans, et vérifier que `mammomesh.dump` n'est pas vide avant de supprimer la base.

```bash
docker compose stop api orchestrator
docker compose run --rm -T --entrypoint restic backup dump latest /mammomesh.dump > mammomesh.dump
docker compose exec -T postgres dropdb -U mammomesh --force mammomesh
docker compose exec -T postgres createdb -U mammomesh mammomesh
docker compose exec -T postgres pg_restore -U mammomesh -d mammomesh --no-owner < mammomesh.dump
docker compose start api orchestrator
```

`latest` peut être remplacé par l'identifiant d'un instantané de la liste.

Depuis la copie (portable perdu, par exemple) : ajouter `-r /copie/mammomesh/backup/restic` après `restic` dans la commande `dump`, même mot de passe.
