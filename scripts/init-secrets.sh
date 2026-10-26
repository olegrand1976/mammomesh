#!/bin/sh
# Génère les secrets de la pile dans ./secrets (une fois, avant le premier démarrage).
# Les fichiers existants ne sont jamais écrasés.
# Administrateur : mot de passe par défaut « admin$1 », à changer à la première connexion
# (l'interface l'exige). ADMIN_PASSWORD_RANDOM=1 en génère un aléatoire à la place.
set -eu
cd "$(dirname "$0")/.."
mkdir -p secrets
chmod 700 secrets
for name in postgres_password mcp_token_pepper admin_password restic_password; do
	f="secrets/$name.txt"
	if [ -f "$f" ]; then
		echo "conservé : $f"
	else
		if [ "$name" = admin_password ] && [ "${ADMIN_PASSWORD_RANDOM:-0}" != 1 ]; then
			printf '%s' 'admin$1' > "$f"
		else
			openssl rand -base64 32 | tr -d '\n=/+' > "$f"
		fi
		chmod 600 "$f"
		echo "créé : $f"
	fi
done
echo
echo "Compte administrateur : ${ADMIN_USERNAME:-admin}, mot de passe dans secrets/admin_password.txt"
echo "(par défaut admin\$1 : l'interface demande de le changer à la première connexion)"
