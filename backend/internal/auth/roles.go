package auth

// Rôles de la section « Sécurité de l'application » de la note de conception.
const (
	RoleAdmin       = "admin"
	RoleScientist   = "responsable_scientifique"
	RoleDataManager = "data_manager"
	RoleRadiologist = "radiologue"
	RoleReader      = "lecteur"
)

// Roles liste les rôles valides.
var Roles = []string{RoleAdmin, RoleScientist, RoleDataManager, RoleRadiologist, RoleReader}

// ValidRole indique si le rôle existe.
func ValidRole(r string) bool {
	for _, v := range Roles {
		if v == r {
			return true
		}
	}
	return false
}

// CanRunCampaigns : lancer, mettre en pause, arrêter une campagne.
func CanRunCampaigns(role string) bool {
	return role == RoleAdmin || role == RoleScientist
}

// CanProposeConfig : créer et vérifier des brouillons de configuration.
func CanProposeConfig(role string) bool {
	return role == RoleAdmin || role == RoleScientist || role == RoleDataManager
}

// CanTestImages : importer des images et y tester des individus (tous sauf le lecteur).
func CanTestImages(role string) bool { return ValidRole(role) && role != RoleReader }

// CanManageTrainings : relancer ou supprimer un entraînement du CNN.
func CanManageTrainings(role string) bool { return role == RoleAdmin || role == RoleScientist }

// CanPurgeImages : supprimer une image et ses tests.
func CanPurgeImages(role string) bool { return role == RoleAdmin || role == RoleDataManager }

// User est l'utilisateur authentifié d'une requête.
type User struct {
	ID          int64  `json:"id"`
	Username    string `json:"username"`
	DisplayName string `json:"display_name"`
	Role        string `json:"role"`
	// MustChangePassword : compte encore au mot de passe par défaut.
	MustChangePassword bool `json:"must_change_password"`
}
