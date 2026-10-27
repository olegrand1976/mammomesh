// Page d'accueil de l'interface connectée (/suivi) : les menus, leur fonctionnement et leurs liens,
// et la différence entre une campagne et un entraînement. Même ordre que la barre latérale (utils/navigation.ts),
// plus « Mon compte » ; un test le vérifie.

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
    cle: 'entrainements',
    titre: 'Entraînements',
    to: '/suivi/entrainements',
    resume: 'Apprendre au CNN à repérer les lésions sur des mammographies d\'exemple, et suivre cet apprentissage en direct.',
    fonctionnement: [
      'Deux onglets, « En cours » et « Finalisés ». Pour chaque entraînement : où il en est, l\'heure de fin prévue, des courbes qui montrent s\'il progresse, et sa note finale sur des images qu\'il n\'a jamais vues.',
      'Boutons pour démarrer un nouvel entraînement, le mettre en pause, le reprendre, le relancer ou le supprimer. La demande est transmise au serveur, qui l\'exécute automatiquement.',
      'Si le serveur s\'arrête (coupure, panne), l\'entraînement repart tout seul de la dernière étape terminée, sans tout recommencer.',
    ],
    liens: [
      { titre: 'CNN et CephaloMesh', to: '/suivi/documentation/cnn-cephalomesh' },
      { titre: 'Jeux d\'entraînement', to: '/suivi/documentation/jeux-entrainement' },
    ],
  },
  {
    cle: 'campagnes',
    titre: 'Campagnes',
    to: '/suivi/campagnes',
    resume: 'Suivre en direct la recherche des meilleurs réglages de CephaloMesh, essai après essai.',
    fonctionnement: [
      'La liste montre chaque campagne : son état, le nombre de générations déjà faites, la meilleure part de cancers trouvés jusqu\'ici (sensibilité, mesurée sur des images de contrôle, E-val) et le temps de calcul déjà utilisé.',
      'L\'écran d\'une campagne se met à jour tout seul : progrès à chaque génération, graphique des meilleurs compromis (cancers trouvés contre rapidité), détail de chaque version testée et historique des événements.',
      'Seuls l\'administrateur et le responsable scientifique peuvent mettre en pause, reprendre, arrêter ou prolonger une campagne, et, une fois finie, la désactiver (elle n\'est plus prise en compte, mais reste consultable) ou la supprimer définitivement. Une campagne terminée peut servir de point de départ à la suivante.',
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
    resume: 'Préparer une campagne, la faire vérifier, puis la lancer.',
    fonctionnement: [
      'Chaque réglage (part minimale de cancers à trouver, nombre de versions par génération, nombre maximal de générations, temps de calcul alloué…) est expliqué, avec ce que change une hausse ou une baisse.',
      'La campagne est d\'abord enregistrée comme brouillon, puis contrôlée (valeurs permises, durée et calcul estimés) avant de pouvoir être lancée.',
      'Le data manager peut préparer et faire vérifier ; seuls l\'administrateur et le responsable scientifique peuvent lancer.',
    ],
    liens: [
      { titre: 'Guide : Nouvelle campagne', to: '/suivi/documentation/nouvelle-campagne' },
      { titre: 'Modèle mathématique', to: '/suivi/documentation/mathematiques' },
    ],
  },
  {
    cle: 'images',
    titre: 'Images',
    to: '/suivi/images',
    resume: 'Ouvrir une mammographie et voir ce qu\'en pense une version de MammoMesh.',
    fonctionnement: [
      'Import de fichiers DICOM, le format des appareils de radiologie. Le nom et l\'identifiant de la patiente sont supprimés à l\'import : seuls l\'image et quelques réglages techniques sont gardés.',
      'Affichage en pleine résolution : zoom, réglage du contraste, inversion noir et blanc, zones signalées.',
      'Test d\'une version choisie dans une campagne : quand le service d\'inférence tourne, le score du CNN entraîné et les zones où il a regardé ; sinon un résultat simulé. L\'écran indique toujours lequel des deux il affiche.',
    ],
    liens: [{ titre: 'Guide : Images', to: '/suivi/documentation/images' }],
  },
  {
    cle: 'documentation',
    titre: 'Documentation',
    to: '/suivi/documentation',
    resume: 'Toutes les explications : comment ça marche, les formules, et un guide pour chaque écran.',
    fonctionnement: [
      'Pour commencer : ce que font le CNN et CephaloMesh, et pourquoi les combiner. Puis le déroulement d\'une campagne, le modèle mathématique avec toutes ses formules, et le lexique des mutations.',
      'Un guide pour chaque écran (Nouvelle campagne, suivi d\'une campagne, Images), les banques d\'images utilisées et leurs conditions d\'usage, un glossaire des sigles et mots techniques, et l\'installation sur Windows 11 (PDF).',
      'Les mêmes textes se trouvent dans le dossier docs/ du projet.',
    ],
    liens: [
      { titre: 'CNN et CephaloMesh', to: '/suivi/documentation/cnn-cephalomesh' },
      { titre: 'Sommaire', to: '/suivi/documentation' },
      { titre: 'Glossaire', to: '/suivi/documentation/glossaire' },
    ],
  },
  {
    cle: 'jetons',
    titre: 'Jetons MCP',
    to: '/suivi/jetons',
    resume: 'Permettre à Claude Code, un assistant d\'intelligence artificielle, de consulter les résultats ou de suggérer des réglages.',
    fonctionnement: [
      'Un jeton est une clé d\'accès personnelle. Un jeton « lecture » permet seulement de consulter les campagnes et leurs résultats.',
      'Un jeton « proposition » permet en plus de préparer des brouillons de campagne ; rien n\'est lancé ni modifié sans la confirmation d\'une personne.',
      'Chaque jeton est valable 90 jours, n\'est montré qu\'une fois, à sa création, et peut être annulé à tout moment.',
    ],
    liens: [],
  },
  {
    cle: 'compte',
    titre: 'Mon compte',
    to: '/suivi/compte',
    resume: 'Votre nom affiché et votre mot de passe (en bas de la barre latérale, en cliquant sur votre nom).',
    fonctionnement: [
      'Changer de mot de passe : 12 caractères au moins.',
      'Le mot de passe donné au départ doit être changé à la première connexion : rien d\'autre n\'est accessible avant.',
    ],
    liens: [],
  },
]

/** Différence entre une campagne et un entraînement, ligne par ligne, expliquée pour un non-spécialiste. */
export const comparaison: { aspect: string; campagne: string; entrainement: string }[] = [
  {
    aspect: 'Ce qu\'on améliore',
    campagne: 'La façon de raisonner. CephaloMesh a des dizaines de réglages : à partir de quel niveau de certitude s\'alerter, quelles vérifications faire (les deux vues, les deux seins, l\'examen précédent), combien de zones faire réexaminer de plus près… Une campagne cherche la meilleure combinaison de ces réglages.',
    entrainement: 'La façon de regarder. Le CNN contient des millions de petits réglages internes, appelés poids, qui décident de ce qu\'il remarque sur une image. Un entraînement ajuste ces poids pour qu\'il reconnaisse ce qui ressemble à une lésion.',
  },
  {
    aspect: 'Comment',
    campagne: 'Par essais et sélection, comme l\'évolution des espèces. Chaque combinaison de réglages est une « version » de MammoMesh, appelée individu. On en essaie beaucoup, on garde les meilleures, on en dérive de nouvelles en changeant un réglage (mutation) ou en mélangeant deux bonnes versions (croisement), et on recommence.',
    entrainement: 'Par l\'exemple, comme un étudiant qui s\'exerce avec un corrigé. On montre au CNN des mammographies dont le diagnostic est connu (prouvé par biopsie). Il donne sa réponse, on la compare au corrigé, et ses poids sont légèrement corrigés pour se tromper un peu moins la fois suivante.',
  },
  {
    aspect: 'Rythme',
    campagne: 'Par générations. Une génération, c\'est un tour complet : 1) on part des meilleures versions gardées jusque-là (les parents) ; 2) on fabrique de nouvelles versions à partir d\'elles ; 3) on teste chacune sur des mammographies (100 versions par génération par défaut) ; 4) on garde les meilleures, qui deviennent les parents du tour suivant. Une campagne enchaîne ainsi des dizaines, voire des centaines de générations (200 au plus par défaut).',
    entrainement: 'Par époques. Une époque est finie quand le CNN a vu une fois chacune des images d\'exemple, en se corrigeant au fur et à mesure. Il en faut plusieurs (15 au plus par défaut) : à chaque passage, il affine ce qu\'il a appris. Après chaque époque, on mesure ses progrès sur des images mises de côté.',
  },
  {
    aspect: 'Ce qu\'on mesure',
    campagne: 'Deux choses à la fois : la part de cancers trouvés (sensibilité) et le temps de calcul par examen. Une version plus sûre est souvent plus lente : on garde donc les meilleurs compromis, de la plus prudente à la plus rapide. Une version qui trouve trop peu de cancers (85 % au minimum par défaut) est écartée d\'office, même si elle est très rapide.',
    entrainement: 'Pendant l\'apprentissage : ses erreurs (la perte, qui doit baisser) et sa capacité à séparer images saines et malades (l\'AUC, de 0,5 pour le hasard à 1 pour la perfection). À la fin, sur des patientes qu\'il n\'a jamais vues : AUC, sensibilité (cancers trouvés) et spécificité (examens sains bien reconnus comme sains).',
  },
  {
    aspect: 'Quand ça s\'arrête',
    campagne: 'Quand le nombre maximal de générations est atteint, quand le temps de calcul alloué est épuisé, ou quand plus aucun progrès n\'apparaît pendant 15 générations (par défaut).',
    entrainement: 'Après le nombre d\'époques prévu, ou plus tôt si le CNN ne progresse plus pendant 5 époques (par défaut) : continuer ne servirait qu\'à lui faire apprendre par cœur les images vues.',
  },
  {
    aspect: 'Résultat',
    campagne: 'Une liste de versions (individus), de la plus prudente à la plus rapide, chacune avec ses mesures et la liste des mutations qui l\'ont produite. Une personne habilitée choisit laquelle retenir.',
    entrainement: 'Un CNN entraîné : un fichier contenant ses poids et le seuil à partir duquel il s\'alerte, avec un rapport de ses résultats. C\'est lui que les campagnes utiliseront ensuite.',
  },
  {
    aspect: 'Où ça tourne',
    campagne: 'Sur le serveur, dans le programme qui pilote les campagnes (l\'orchestrateur). Il tourne seul, jour et nuit, et reprend où il en était après un redémarrage.',
    entrainement: 'Sur la carte graphique (GPU) du serveur, faite pour les calculs lourds sur les images. Si le serveur s\'arrête, l\'entraînement reprend seul à la dernière époque terminée.',
  },
  {
    aspect: 'État actuel',
    campagne: 'En rodage : les campagnes tournent, mais leurs mesures sont simulées, donc sans valeur, tant que le CNN entraîné n\'y est pas branché.',
    entrainement: 'Réel : un premier CNN (EfficientNet-B0) apprend sur la banque d\'images CMMD (1 775 patientes, diagnostic prouvé par biopsie).',
  },
]
