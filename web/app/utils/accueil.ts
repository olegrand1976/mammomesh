// Page d'accueil de l'interface connectée (/suivi) : les menus, leur fonctionnement et leurs liens,
// et la différence entre une campagne et un entraînement. Un test vérifie que chaque lien du menu y figure.

export interface Lien { titre: string; to: string }

export interface Menu {
  cle: string
  titre: string
  to: string
  /** Permission (session.permissions) nécessaire pour voir le menu ; sans : ouvert à tous les rôles. */
  permission?: 'proposer_config' | 'gerer_entrainements' | 'tester_images' | 'lancer_campagnes'
  resume: string
  fonctionnement: string[]
  liens: Lien[]
}

export const menus: Menu[] = [
  {
    cle: 'campagnes',
    titre: 'Campagnes',
    to: '/suivi/campagnes',
    resume: 'La recherche du meilleur raisonnement CephaloMesh, génération après génération, suivie en direct.',
    fonctionnement: [
      'La liste montre chaque campagne : état, génération atteinte, meilleure sensibilité en validation (E-val), budget consommé.',
      'L\'écran d\'une campagne se met à jour en direct : progrès par génération, front de Pareto (sensibilité contre temps de calcul), fiche d\'un individu, journal.',
      'Pause, reprise, arrêt et hausse de budget sont réservés à l\'administrateur et au responsable scientifique ; une campagne finie peut servir de départ à une nouvelle.',
    ],
    liens: [
      { titre: 'Guide de l\'écran de suivi', to: '/suivi/documentation/suivi-campagne' },
      { titre: 'Fonctionnement d\'une campagne', to: '/suivi/documentation/fonctionnement' },
      { titre: 'Lexique des mutations', to: '/suivi/documentation/mutations' },
    ],
  },
  {
    cle: 'nouvelle',
    titre: 'Nouvelle campagne',
    to: '/suivi/nouvelle',
    permission: 'proposer_config',
    resume: 'Préparer, vérifier puis lancer une campagne.',
    fonctionnement: [
      'Chaque réglage (plancher de sensibilité, population, plafond de générations, budget, graine…) a son aide et l\'effet calculé d\'une hausse ou d\'une baisse.',
      'La configuration est figée en brouillon, puis vérifiée (plages, durée et calcul estimés) avant tout lancement.',
      'Le lancement revient à l\'administrateur ou au responsable scientifique ; le data manager peut préparer et vérifier.',
    ],
    liens: [
      { titre: 'Guide : Nouvelle campagne', to: '/suivi/documentation/nouvelle-campagne' },
      { titre: 'Modèle mathématique', to: '/suivi/documentation/mathematiques' },
    ],
  },
  {
    cle: 'entrainements',
    titre: 'Entraînements',
    to: '/suivi/entrainements',
    resume: 'L\'apprentissage du CNN sur la banque d\'images, suivi en direct, avec ses temps de fin estimés.',
    fonctionnement: [
      'Onglets « En cours » et « Finalisés » ; pour chacun : phase, avancement, fin estimée, courbes d\'AUC et de perte, résultat final en validation.',
      'Nouvel entraînement, pause (en fin d\'époque ou tout de suite), reprise, relance et suppression : l\'interface dépose une demande, le veilleur du serveur l\'exécute.',
      'Un entraînement coupé (machine éteinte, plantage) reprend seul à sa dernière époque finie.',
    ],
    liens: [
      { titre: 'Jeux d\'entraînement', to: '/suivi/documentation/jeux-entrainement' },
    ],
  },
  {
    cle: 'images',
    titre: 'Images',
    to: '/suivi/images',
    resume: 'Importer une mammographie, la lire et y tester une version du couple CNN + CephaloMesh.',
    fonctionnement: [
      'Import DICOM pseudonymisé : seuls les pixels et des données techniques sont gardés, jamais le nom ni l\'identifiant de la patiente.',
      'Visionneuse en pleine résolution : zoom, fenêtrage, inversion, zones signalées.',
      'Test d\'un individu choisi dans une campagne : score de l\'image et zones. Tant que le CNN n\'est pas branché, le résultat est simulé et le dit.',
    ],
    liens: [{ titre: 'Guide : Images', to: '/suivi/documentation/images' }],
  },
  {
    cle: 'jetons',
    titre: 'Jetons MCP',
    to: '/suivi/jetons',
    resume: 'Brancher Claude Code sur le serveur, pour interroger les résultats ou proposer des réglages.',
    fonctionnement: [
      'Un jeton « lecture » donne accès aux campagnes, générations, individus et rapports.',
      'Un jeton « proposition » permet en plus de préparer des brouillons ; lancer ou régler une campagne demande toujours une confirmation humaine.',
      'Jetons valables 90 jours, affichés une seule fois, révocables à tout moment.',
    ],
    liens: [],
  },
  {
    cle: 'documentation',
    titre: 'Documentation',
    to: '/suivi/documentation',
    resume: 'Le fonctionnement complet, les formules et un guide pour chaque écran.',
    fonctionnement: [
      'Fonctionnement d\'une campagne de bout en bout, modèle mathématique (toutes les formules), lexique des mutations.',
      'Guides d\'écran : Nouvelle campagne, suivi d\'une campagne, Images ; jeux d\'entraînement et leurs licences.',
      'Le même contenu se trouve dans le dossier docs/ du dépôt.',
    ],
    liens: [{ titre: 'Sommaire', to: '/suivi/documentation' }],
  },
  {
    cle: 'compte',
    titre: 'Mon compte',
    to: '/suivi/compte',
    resume: 'Votre nom affiché et votre mot de passe (en haut à droite, sur votre nom).',
    fonctionnement: [
      'Changer de mot de passe : 12 caractères au moins.',
      'Le mot de passe par défaut doit être changé à la première connexion : rien d\'autre n\'est accessible avant.',
    ],
    liens: [],
  },
]

/** Différence entre une campagne et un entraînement, ligne par ligne. */
export const comparaison: { aspect: string; campagne: string; entrainement: string }[] = [
  { aspect: 'Ce qu\'on cherche', campagne: 'Le meilleur raisonnement : les réglages de CephaloMesh (seuils, contrôles, budget d\'activité…).', entrainement: 'Le meilleur regard : les millions de poids du CNN, qui lit les images.' },
  { aspect: 'Comment', campagne: 'Évolution : une population d\'individus mute, les meilleurs compromis sont gardés à chaque génération.', entrainement: 'Apprentissage : à chaque image, les poids sont corrigés pour se tromper un peu moins (descente de gradient).' },
  { aspect: 'Unité de progrès', campagne: 'La génération.', entrainement: 'L\'époque (un passage sur toutes les images d\'entraînement).' },
  { aspect: 'Ce qu\'on mesure', campagne: 'Sensibilité (E-ajust, E-val) et temps de calcul : front de Pareto, sous un plancher de sensibilité.', entrainement: 'Perte et AUC en ajustement ; à la fin, AUC, sensibilité et spécificité en validation.' },
  { aspect: 'Résultat', campagne: 'Des individus : couples CNN + CephaloMesh, du plus sensible au plus rapide.', entrainement: 'Un modèle (fichier de poids) et son seuil, rangés dans modeles/.' },
  { aspect: 'Où ça tourne', campagne: 'L\'orchestrateur, dans la pile Docker.', entrainement: 'Le moteur Python sur le GPU du serveur, lancé par le veilleur.' },
  { aspect: 'État actuel', campagne: 'Boucle simulée (sans valeur) tant que le CNN entraîné n\'y est pas branché.', entrainement: 'Réel : premier CNN (EfficientNet-B0) sur la banque CMMD.' },
]
