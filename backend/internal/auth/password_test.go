package auth

import (
	"bytes"
	"testing"
)

func TestHashAndCheckPassword(t *testing.T) {
	h, err := HashPassword("un mot de passe assez long")
	if err != nil {
		t.Fatal(err)
	}
	if ok, err := CheckPassword(h, "un mot de passe assez long"); err != nil || !ok {
		t.Fatalf("mot de passe correct refusé : %v %v", ok, err)
	}
	if ok, _ := CheckPassword(h, "un autre mot de passe"); ok {
		t.Fatal("mot de passe faux accepté")
	}
	h2, _ := HashPassword("un mot de passe assez long")
	if h == h2 {
		t.Fatal("deux empreintes identiques : le sel ne varie pas")
	}
}

func TestShortPasswordRejected(t *testing.T) {
	if _, err := HashPassword("court"); err == nil {
		t.Fatal("mot de passe trop court accepté")
	}
}

func TestHashTokenDependsOnPepper(t *testing.T) {
	if bytes.Equal(HashToken("jeton", "a"), HashToken("jeton", "b")) {
		t.Fatal("le poivre n'influence pas l'empreinte")
	}
}

func TestRoles(t *testing.T) {
	if !CanRunCampaigns(RoleScientist) || CanRunCampaigns(RoleReader) || CanRunCampaigns(RoleDataManager) {
		t.Fatal("droits de lancement incorrects")
	}
	if !CanProposeConfig(RoleDataManager) || CanProposeConfig(RoleRadiologist) {
		t.Fatal("droits de proposition incorrects")
	}
}

func TestDefaultPassword(t *testing.T) {
	if _, err := HashPassword(DefaultAdminPassword); err != ErrWeakPassword {
		t.Fatalf("le mot de passe par défaut ne doit pas passer la règle : %v", err)
	}
	h, err := HashDefaultPassword()
	if err != nil {
		t.Fatal(err)
	}
	if ok, _ := CheckPassword(h, DefaultAdminPassword); !ok {
		t.Fatal("empreinte du mot de passe par défaut invalide")
	}
	// La longueur se compte en caractères, pas en octets.
	if _, err := HashPassword("éééééééééée"); err != ErrWeakPassword {
		t.Fatalf("11 caractères accentués acceptés : %v", err)
	}
}
