# Sauvegarde quotidienne de PostgreSQL dans un dépôt restic (profil compose « backup »).
# Même image que le serveur : pg_dump a sa version majeure.
FROM postgres:18-alpine
RUN apk add --no-cache restic tzdata
# Heure de BACKUP_HOUR ; TZ se change dans l'environnement du service.
ENV TZ=Europe/Brussels
COPY --chmod=755 sauvegarde.sh /usr/local/bin/sauvegarde
ENTRYPOINT ["/usr/local/bin/sauvegarde"]
CMD ["quotidienne"]
