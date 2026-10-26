// Fonctionnement complet d'une campagne : source unique de la page /suivi/fonctionnement
// et de docs/fonctionnement-campagne.md (un test vérifie la concordance).

export interface Bloc {
  p?: string
  etapes?: string[] // liste numérotée
  points?: string[] // liste à puces
  table?: { entetes: string[]; lignes: string[][] }
}

export interface Partie {
  id: string
  titre: string
  blocs: Bloc[]
}

export const parties: Partie[] = [
  {
    id: 'en-bref',
    titre: 'En bref',
    blocs: [
      { p: 'Une campagne fait évoluer une population de configurations CephaloMesh, le raisonnement qui suit le CNN, pour trouver les meilleurs compromis entre sensibilité et temps de calcul, sous un plancher de sensibilité. Elle tourne seule, génération après génération, jusqu\'à un critère d\'arrêt fixé avant le lancement. C\'est un outil de recherche : aucun résultat ne sert à un diagnostic.' },
      { p: 'Dans cette version du socle, la boucle est simulée. L\'orchestrateur, les règles de réalisabilité, de sélection et d\'arrêt, le temps réel, le journal et l\'audit sont réels ; les mesures (sensibilités, coûts, génomes) viennent d\'une simulation déterministe, en attendant le branchement du CNN et de CephaloMesh.' },
    ],
  },
  {
    id: 'acteurs',
    titre: 'Qui fait quoi',
    blocs: [
      {
        table: {
          entetes: ['Composant', 'Rôle'],
          lignes: [
            ['Interface', 'Prépare, vérifie et lance les campagnes, les suit en direct et les pilote. Elle peut être fermée : rien ne s\'arrête.'],
            ['API', 'Vérifie les droits et les contrôles, enregistre les brouillons, relaie le temps réel aux navigateurs.'],
            ['Orchestrateur', 'Seul à calculer. Une seule instance active à la fois (verrou en base) ; il regarde chaque seconde s\'il a une campagne à faire avancer.'],
            ['PostgreSQL', 'Garde configurations, générations, individus, journal et audit. Chaque génération y est écrite en une seule transaction.'],
            ['Redis', 'Compte les tentatives de connexion ; s\'il manque, des compteurs en mémoire prennent le relais.'],
            ['Claude Code (MCP)', 'Lit les résultats, propose des brouillons ; lancer ou régler une campagne demande toujours une confirmation humaine.'],
          ],
        },
      },
    ],
  },
  {
    id: 'cycle',
    titre: 'Cycle de vie',
    blocs: [
      {
        etapes: [
          'Brouillon : la configuration est figée et reçoit une empreinte (SHA-256). La même configuration donne toujours la même empreinte.',
          'Vérification : chaque réglage est comparé à sa plage ; la durée et le calcul sont estimés. Rien n\'est lancé.',
          'Lancement : par le responsable scientifique ou l\'administrateur, une seule fois par brouillon. La campagne passe « En cours ».',
          'En cours ⇄ En pause : la pause prend effet à la fin du lot en cours ; à la reprise, la génération interrompue est refaite à l\'identique.',
          'Fin : « Terminée » quand un critère d\'arrêt est atteint, « Arrêtée » sur demande d\'une personne, « En échec » si l\'orchestrateur rencontre une erreur (détaillée au journal).',
        ],
      },
    ],
  },
  {
    id: 'generation',
    titre: 'Une génération, pas à pas',
    blocs: [
      {
        etapes: [
          'Avant de commencer, l\'orchestrateur vérifie les critères d\'arrêt : plafond de générations, budget atteint, stagnation de E-val.',
          'Le calcul avance par lots. À chaque lot, il publie la progression (lot a sur b, budget consommé) et vérifie que la campagne est toujours en cours.',
          'Population : chaque individu est l\'enfant d\'un individu du front précédent, tiré au hasard, avec m = 1 + Poisson(μ) mutations, plafonné par « Mutations au plus ». À la première génération, les individus sont des fondateurs.',
          'Mesures de chaque individu : sensibilité sur E-ajust avec son intervalle (bornes basse et haute), sensibilité sur E-val, coût par examen en millisecondes.',
          'Réalisabilité : un individu est retenu si la borne basse de sa sensibilité atteint « plancher − marge ». Les autres sont écartés, et le journal le dit.',
          'Front de Pareto : parmi les réalisables, ceux qu\'aucun autre ne bat à la fois en sensibilité et en coût. Ils deviennent les parents de la génération suivante.',
          'Meilleur : le réalisable de meilleure sensibilité sur E-val. S\'il dépasse le record de la campagne d\'au moins 0,1 point, le journal annonce un nouveau meilleur et le compteur de stagnation revient à 0 ; sinon le compteur augmente, avec une alerte 3 générations avant l\'arrêt.',
          'Écriture : génération, individus, compteurs, budget et événements du journal sont écrits en une seule transaction ; aucun écran ne voit une génération à moitié écrite.',
          'Diffusion : chaque événement est notifié par PostgreSQL et relayé aux navigateurs ouverts, qui se mettent à jour seuls.',
        ],
      },
    ],
  },
  {
    id: 'e-ajust-e-val',
    titre: 'Deux mesures : E-ajust et E-val',
    blocs: [
      { p: 'Les patientes du jeu d\'évolution sont coupées une fois pour toutes en deux parties : E-ajust (75 %) et E-val (25 %). L\'évolution n\'optimise que sur E-ajust : la réalisabilité et le front de Pareto s\'y calculent.' },
      { p: 'E-val ne sert jamais à optimiser. Il désigne le meilleur individu, arrête la campagne quand il ne progresse plus et choisira le front exporté. Un individu sélectionné parmi des milliers paraît toujours un peu meilleur sur les données qui ont servi à le choisir : l\'écart E-ajust − E-val mesure cet optimisme. Avec environ 20 000 configurations testées, la note de conception l\'estime à 2 à 3 points de sensibilité.' },
    ],
  },
  {
    id: 'arrets',
    titre: 'Critères d\'arrêt',
    blocs: [
      {
        table: {
          entetes: ['Critère', 'Règle', 'Réglage'],
          lignes: [
            ['Plafond', 'Le nombre de générations atteint le plafond.', 'Générations au plus'],
            ['Budget', 'Les cœur-heures consommées atteignent le budget ; la génération commencée va jusqu\'au bout, la dernière peut donc le dépasser un peu.', 'Budget (cœur-heures), augmentable en cours de route'],
            ['Stagnation', 'N générations de suite sans nouveau meilleur sur E-val (gain d\'au moins 0,1 point). Le compteur tourne aussi tant qu\'aucun individu n\'est réalisable.', 'Arrêt si E-val stable'],
            ['Arrêt humain', '« Arrêter » dans le suivi, ou l\'outil MCP campagne_reglage après confirmation.', '—'],
          ],
        },
      },
      { p: 'Le premier critère atteint l\'emporte ; le journal indique lequel.' },
    ],
  },
  {
    id: 'reproductibilite',
    titre: 'Reproductibilité, reprise et traçabilité',
    blocs: [
      {
        points: [
          'Le tirage dépend de la graine, du numéro de campagne et du numéro de génération : une génération recalculée donne exactement le même résultat.',
          'Après un redémarrage du serveur, l\'orchestrateur reprend chaque campagne en cours à la génération suivante et l\'annonce au journal ; rien de ce qui est écrit n\'est recalculé.',
          'La configuration d\'une campagne ne change jamais après le lancement ; son empreinte l\'identifie. Seul le budget peut être augmenté.',
          'Chaque geste (lancement, pause, reprise, budget, arrêt, appel MCP) est inscrit au journal d\'audit, avec son auteur et sa source.',
        ],
      },
    ],
  },
  {
    id: 'temps-reel',
    titre: 'Temps réel',
    blocs: [
      {
        points: [
          'Les événements du journal arrivent tous, dans l\'ordre, sans fusion.',
          'La progression des lots est fusionnée : au plus un message toutes les 2 secondes par campagne, pour ne pas saturer les navigateurs.',
          'Après une coupure, le navigateur se reconnecte et reprend après le dernier événement reçu ; si trop d\'événements ont été manqués, l\'écran relit toutes ses données de lui-même.',
        ],
      },
    ],
  },
  {
    id: 'vraie-boucle',
    titre: 'Ce qui changera avec la vraie boucle',
    blocs: [
      {
        points: [
          'Deux vitesses : le CNN évolue lentement (des heures par évaluation), CephaloMesh rapidement (0,2 à 0,5 ms par examen), sur des sorties du CNN mises en cache.',
          'Côté CNN, les enfants montent par paliers asynchrones (un pli partiel, un pli, tous les plis) ; seuls les meilleurs vont plus loin.',
          'La population CephaloMesh d\'un nouveau CNN démarre depuis l\'archive de son parent.',
          'Sélection NSGA-II sur les couples CNN + CephaloMesh, avec deux objectifs (sensibilité, coût) et des contraintes, dont la non-infériorité par type de lésion.',
          'Îles, choix adaptatif des opérateurs de mutation et bascule vers des macro-mutations quand le front ne bouge plus sur E-val pendant 8 générations.',
        ],
      },
    ],
  },
]
