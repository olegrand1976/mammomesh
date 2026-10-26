// Package auth gère les mots de passe, les sessions, les jetons MCP et les rôles.
package auth

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"
)

// Paramètres Argon2id (recommandation OWASP : 19 Mio, 2 passes, 1 fil au minimum).
const (
	argonTime    = 2
	argonMemory  = 64 * 1024
	argonThreads = 2
	argonKeyLen  = 32
	saltLen      = 16
)

var errBadHash = errors.New("empreinte de mot de passe invalide")

// DefaultAdminPassword est le mot de passe du premier administrateur quand aucun n'est fourni.
// Il échappe à la règle des 12 caractères, mais le compte doit le changer à la première connexion.
const DefaultAdminPassword = "admin$1"

// MinPasswordLen est la longueur minimale d'un mot de passe choisi.
const MinPasswordLen = 12

// ErrWeakPassword signale un mot de passe trop court.
var ErrWeakPassword = errors.New("le mot de passe doit compter au moins 12 caractères")

// HashPassword renvoie une empreinte au format PHC : $argon2id$v=19$m=…,t=…,p=…$sel$clé.
func HashPassword(password string) (string, error) {
	if len([]rune(password)) < MinPasswordLen {
		return "", ErrWeakPassword
	}
	return hashPassword(password)
}

// HashDefaultPassword hache le mot de passe par défaut, sans la règle de longueur.
func HashDefaultPassword() (string, error) { return hashPassword(DefaultAdminPassword) }

func hashPassword(password string) (string, error) {
	salt := make([]byte, saltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	key := argon2.IDKey([]byte(password), salt, argonTime, argonMemory, argonThreads, argonKeyLen)
	b64 := base64.RawStdEncoding
	return fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version, argonMemory, argonTime, argonThreads, b64.EncodeToString(salt), b64.EncodeToString(key)), nil
}

// CheckPassword compare en temps constant.
func CheckPassword(encoded, password string) (bool, error) {
	parts := strings.Split(encoded, "$")
	if len(parts) != 6 || parts[1] != "argon2id" {
		return false, errBadHash
	}
	var version int
	if _, err := fmt.Sscanf(parts[2], "v=%d", &version); err != nil || version != argon2.Version {
		return false, errBadHash
	}
	var mem uint32
	var t uint32
	var p uint8
	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &mem, &t, &p); err != nil {
		return false, errBadHash
	}
	b64 := base64.RawStdEncoding
	salt, err := b64.DecodeString(parts[4])
	if err != nil {
		return false, errBadHash
	}
	want, err := b64.DecodeString(parts[5])
	if err != nil {
		return false, errBadHash
	}
	got := argon2.IDKey([]byte(password), salt, t, mem, p, uint32(len(want)))
	return subtle.ConstantTimeCompare(got, want) == 1, nil
}

// NewToken renvoie un jeton aléatoire de 32 octets, encodé pour une URL ou un en-tête.
func NewToken(prefix string) (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return prefix + base64.RawURLEncoding.EncodeToString(b), nil
}

// HashToken garde en base l'empreinte du jeton, jamais le jeton lui-même.
// Le poivre (secret du serveur) rend une fuite de la table inexploitable seule.
func HashToken(token, pepper string) []byte {
	m := hmac.New(sha256.New, []byte(pepper))
	m.Write([]byte(token))
	return m.Sum(nil)
}
