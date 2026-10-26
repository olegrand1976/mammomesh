#!/bin/sh
# Génère les secrets de la pile dans ./secrets (une fois, avant le premier démarrage).
# Les fichiers existants ne sont jamais écrasés.
set -eu
cd "$(dirname "$0")/.."
mkdir -p secrets
chmod 700 secrets
for name in postgres_password mcp_token_pepper admin_password restic_password; do
	f="secrets/$name.txt"
	if [ -f "$f" ]; then
		echo "conservé : $f"
	else
		openssl rand -base64 32 | tr -d '\n=/+' > "$f"
		chmod 600 "$f"
		echo "créé : $f"
	fi
done
echo
echo "Compte administrateur : ${ADMIN_USERNAME:-admin}, mot de passe dans secrets/admin_password.txt"
