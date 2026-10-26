// Guide de l'écran de suivi d'une campagne (/suivi/campagnes/:id) : chaque zone, ses éléments et leur rôle.
// Source unique pour l'aide affichée dans l'écran ; docs/guide-suivi-campagne.md en reprend le contenu
// (un test vérifie la concordance).

export interface ElementGuide {
  nom: string
  description: string
}

export interface ZoneGuide {
  cle: 'bandeau' | 'progres' | 'front' | 'fiche' | 'journal'
  titre: string
  resume: string
  elements: ElementGuide[]
  aSavoir?: string
}

export const zones: ZoneGuide[] = [
  {
    cle: 'bandeau',
    titre: 'Bandeau de la campagne',
    resume: 'L\'état de la campagne en un coup d\'œil, et les commandes pour la piloter.',
    elements: [
      { nom: 'Nom et badge d\'état', description: 'En cours, En pause, Terminée (plafond, budget ou stagnation atteint), Arrêtée (par une personne) ou En échec (erreur de l\'orchestrateur, voir le journal).' },
      { nom: 'En direct / Reconnexion…', description: 'Pastille verte : le flux temps réel est ouvert, l\'écran se met à jour seul. Orange : connexion perdue ; le navigateur se reconnecte et reprend là où il s\'était arrêté, sans rien perdre ni recharger.' },
      { nom: 'Pause, Reprendre', description: 'Pause suspend la campagne après le lot en cours ; la génération interrompue sera refaite à l\'identique à la reprise (même graine, même tirage).' },
      { nom: 'Budget +', description: 'Augmente le budget de calcul, en cœur-heures, par exemple quand la campagne approche de son plafond alors qu\'elle progresse encore. Le budget ne se baisse jamais.' },
      { nom: 'Repartir de cette campagne', description: 'Sur une campagne terminée ou arrêtée : ouvre « Nouvelle campagne » avec ses réglages, à modifier, et le front de sa génération finale comme parents. Une génération intermédiaire peut être choisie à la place.' },
      { nom: 'Arrêter', description: 'Arrêt définitif, après confirmation. Les résultats déjà calculés restent consultables ; la campagne ne peut plus reprendre.' },
      { nom: 'Génération', description: 'Générations terminées sur le plafond, avec une barre. Pendant le calcul, la ligne « génération N : lot a sur b » suit la génération en cours, lot par lot.' },
      { nom: 'Budget', description: 'Cœur-heures consommées sur le budget, mis à jour à chaque lot. Quand la barre est pleine, la campagne s\'arrête à la fin de la génération en cours.' },
      { nom: 'Meilleure sensibilité (E-val)', description: 'Meilleure sensibilité mesurée sur E-val parmi les individus réalisables depuis le début ; « — » tant qu\'aucun ne l\'est. Dessous, le nombre de générations sans progrès d\'au moins 0,1 point : la campagne s\'arrête quand il atteint le réglage « Arrêt si E-val stable ».' },
      { nom: 'Fin estimée', description: 'Durée moyenne des 10 dernières générations × générations restantes, « au plus » car la stagnation ou le budget peuvent arrêter avant. « — » hors calcul.' },
      { nom: 'Lignée et configuration', description: 'Préfixe des noms d\'individus et 8 premiers caractères de l\'empreinte de la configuration figée : deux campagnes à la même empreinte ont exactement les mêmes réglages.' },
      { nom: 'Départ', description: 'Pour une campagne qui repart d\'une autre : lien vers la campagne d\'origine et génération dont le front a fourni les parents.' },
    ],
    aSavoir: 'Les commandes n\'apparaissent qu\'au responsable scientifique et à l\'administrateur ; chaque geste est inscrit au journal d\'audit. L\'interface peut être fermée : l\'orchestrateur continue seul.',
  },
  {
    cle: 'progres',
    titre: 'Progrès par génération',
    resume: 'Comment la recherche avance, génération après génération.',
    elements: [
      { nom: 'Meilleur (E-val), trait bleu', description: 'Sensibilité sur E-val du meilleur individu réalisable de chaque génération. Un trou signifie qu\'aucun individu n\'était réalisable (l\'infobulle indique « aucun réalisable »).' },
      { nom: 'Médiane, trait gris', description: 'Sensibilité médiane (E-ajust) de toute la population, réalisables ou non : elle montre si l\'ensemble progresse, pas seulement le meilleur.' },
      { nom: 'Écart E-ajust − E-val, pointillé orange, axe de droite', description: 'Écart moyen, en points, entre la sensibilité sur laquelle l\'évolution optimise (E-ajust) et celle mesurée à part (E-val), pour les réalisables. C\'est l\'optimisme de la sélection : s\'il grandit, l\'évolution s\'ajuste à E-ajust plus qu\'elle ne progresse vraiment.' },
      { nom: 'Ligne « plancher », tirets', description: 'Le plancher de sensibilité de la configuration. La règle de réalisabilité tolère la marge (seuil = plancher − marge) : un meilleur sous la ligne reste admis.' },
      { nom: 'Infobulle', description: 'Survoler une génération affiche ses trois valeurs.' },
    ],
    aSavoir: 'E-val ne sert jamais à optimiser : seulement à arrêter la campagne et à choisir le front exporté. Une courbe bleue plate depuis longtemps annonce l\'arrêt par stagnation.',
  },
  {
    cle: 'front',
    titre: 'Front de Pareto',
    resume: 'Les meilleurs compromis entre sensibilité et coût de calcul, à la dernière génération.',
    elements: [
      { nom: 'Axes', description: 'Horizontal : coût par examen, en millisecondes (à gauche, plus rapide). Vertical : sensibilité sur E-ajust (en haut, plus sensible). Le coin haut-gauche est l\'idéal.' },
      { nom: 'Front, génération N, en bleu', description: 'Individus réalisables non dominés : aucun autre n\'est à la fois plus sensible et moins coûteux. L\'escalier relie ces compromis, du plus rapide au plus sensible.' },
      { nom: 'Front précédent, en gris', description: 'Le front de la génération d\'avant. Quand le bleu passe au-dessus ou à gauche du gris, la génération a amélioré les compromis.' },
      { nom: 'Clic sur un point', description: 'Ouvre la fiche de l\'individu sous les graphiques. L\'infobulle donne d\'abord son nom, sa sensibilité et son coût.' },
      { nom: 'Liste des individus du front', description: 'Les mêmes individus en tableau (nom, sensibilité, coût), avec un lien « Fiche » : utile au clavier, pour un lecteur d\'écran ou quand deux points se chevauchent.' },
      { nom: 'Message « Aucun individu réalisable »', description: 'Affiché tant qu\'aucun individu n\'atteint « plancher − marge » : le front se dessine dès le premier réalisable.' },
    ],
  },
  {
    cle: 'fiche',
    titre: 'Fiche d\'un individu',
    resume: 'Tout ce qu\'on sait d\'un individu choisi sur le front.',
    elements: [
      { nom: 'Sensibilité (E-ajust) [basse ; haute]', description: 'Sensibilité sur laquelle l\'évolution optimise, avec son intervalle. La borne basse décide de la réalisabilité.' },
      { nom: 'Sensibilité (E-val)', description: 'Même mesure sur la partie de contrôle, jamais utilisée pour optimiser.' },
      { nom: 'Coût par examen', description: 'Temps de calcul de CephaloMesh pour un examen, en millisecondes.' },
      { nom: 'Mutations', description: 'Mutations qui ont produit cet individu, séparées par ›. Chaque code est expliqué en clair juste dessous (gène modifié, partie CephaloMesh ou CNN, effet attendu) ; survoler un code ou le sélectionner au clavier affiche aussi son explication. Le lexique complet est en bas de la fiche.' },
      { nom: 'Parent', description: 'Individu du front précédent dont il descend ; « fondateur » à la première génération.' },
      { nom: 'Gènes', description: 'Valeurs du génome CephaloMesh : seuil, tau, lambda, budget d\'activité, rayon des stimuli (mm), tours (1 ou 2 retours au CNN) et contrôles actifs.' },
      { nom: 'Fermer', description: 'Referme la fiche ; un autre clic sur le front en ouvre une autre.' },
    ],
    aSavoir: 'Dans ce socle, génomes et mesures sont simulés : leur forme est celle de la vraie boucle, pas leurs valeurs.',
  },
  {
    cle: 'journal',
    titre: 'Journal',
    resume: 'Ce qui s\'est passé, en phrases, le plus récent en tête.',
    elements: [
      { nom: 'Lignes', description: 'Heure et message de chaque événement, sans regroupement : lancement, génération terminée, nouveau meilleur, individus écartés, stagnation, changement d\'état, hausse de budget, reprise après redémarrage, fin ou échec.' },
      { nom: 'Taux de rappel dans « nouveau meilleur »', description: 'Toujours la même valeur pendant une campagne : c\'est le taux de rappel cible de la configuration, auquel la sensibilité est mesurée, pas un résultat. Les modèles se comparent à taux de rappel égal.' },
      { nom: 'Filtre', description: 'Restreint la liste à un type : nouveaux meilleurs, générations, individus écartés, stagnation, changements d\'état. « Tous » rétablit tout.' },
      { nom: 'Codes de mutations', description: 'Dans les lignes « nouveau meilleur », chaque code de mutation s\'explique au survol ou au clavier ; le « Lexique des mutations » en bas du journal les décrit tous.' },
      { nom: 'Mises à jour', description: 'Les nouveaux événements s\'insèrent en tête en direct ; les 200 plus récents sont affichés.' },
    ],
    aSavoir: 'Le journal est aussi lisible par Claude Code (outil MCP journal_lire), avec les mêmes messages.',
  },
]

export const zoneParCle = Object.fromEntries(zones.map(z => [z.cle, z])) as Record<ZoneGuide['cle'], ZoneGuide>
