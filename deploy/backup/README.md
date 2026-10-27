# Sauvegarde (profil `backup`)

Chaque jour à `BACKUP_HOUR` h (heure de Bruxelles, `TZ` pour une autre), `pg_dump` de la base `mammomesh` dans le dépôt restic `BACKUP_DIR/restic`, chiffré par `secrets/restic_password.txt`. Au premier démarrage, une sauvegarde est faite tout de suite. Rétention : 7 quotidiennes, 4 hebdomadaires, 12 mensuelles (`GARDER_JOURS`, `GARDER_SEMAINES`, `GARDER_MOIS`). Les images (SeaweedFS) ne sont pas comprises.

`BACKUP_DIR` doit être hors du serveur (disque externe, NAS monté sur l'hôte). Sans `secrets/restic_password.txt`, le dépôt est illisible : garder ce fichier ailleurs aussi.

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
