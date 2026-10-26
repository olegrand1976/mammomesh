// Guide de l'écran « Nouvelle campagne » : rôle et impact de chaque réglage.
// Source unique pour l'aide affichée dans l'écran ; docs/guide-nouvelle-campagne.md en reprend le contenu
// (un test vérifie que chaque réglage y figure). Les exemples chiffrés viennent de la simulation
// actuelle (campagne n° 1, autres réglages par défaut) : ils changeront avec la vraie boucle CNN + CephaloMesh.
import type { CampaignConfig } from '~/types'

export type Section = 'projet' | 'evolution' | 'simulation'

export interface ChampGuide {
  cle: string // notation pointée, comme dans les contrôles et les différences : « evolution.population »
  section: Section
  libelle: string
  plage?: [number, number]
  unite?: string
  format?: string // saisie attendue quand il n'y a pas de plage numérique
  role: string // à quoi sert le réglage
  impact: string // ce que change une hausse ou une baisse, aujourd'hui
  boucleReelle?: string // effet prévu quand le CNN et CephaloMesh seront branchés
}

export const sections: Record<Section, { titre: string; intro: string }> = {
  projet: {
    titre: 'Projet',
    intro: 'Ce que la campagne cherche et les garde-fous écrits avant de voir les résultats.',
  },
  evolution: {
    titre: 'Évolution',
    intro: 'Taille de la recherche, conditions d\'arrêt et manière de produire les enfants.',
  },
  simulation: {
    titre: 'Simulation',
    intro: 'Rythme de la boucle simulée de ce socle. Aucun effet sur les résultats.',
  },
}

export const champs: ChampGuide[] = [
  {
    cle: 'projet.nom', section: 'projet', libelle: 'Nom', format: 'texte, obligatoire',
    role: 'Nom affiché dans la liste des campagnes, le bandeau de suivi, le journal et les réponses MCP. Obligatoire.',
    impact: 'Aucun effet sur les calculs. Il entre dans l\'empreinte de la configuration : deux campagnes aux réglages identiques mais aux noms différents ont deux empreintes.',
  },
  {
    cle: 'projet.objectif', section: 'projet', libelle: 'Objectif en une phrase', format: 'texte libre',
    role: 'L\'hypothèse ou la question, écrite avant le lancement et figée avec la configuration : on ne la réécrit pas après avoir vu les résultats.',
    impact: 'Aucun effet sur les calculs ; change l\'empreinte.',
  },
  {
    cle: 'projet.lignee', section: 'projet', libelle: 'Lignée (majuscules)', format: '3 à 12 lettres A à Z',
    role: 'Préfixe du nom de chaque individu : ORION-G03.02 désigne l\'individu 2 de la génération 3 de la lignée ORION. De 3 à 12 lettres majuscules, sans accent.',
    impact: 'Aucun effet sur les calculs. Une lignée par campagne évite de confondre leurs individus quand on les cite (fiche, comparaison, outils MCP par nom).',
  },
  {
    cle: 'projet.plancher_sensibilite', section: 'projet', libelle: 'Plancher de sensibilité', plage: [0.8, 0.99],
    role: 'Sensibilité minimale, non négociable. Un individu n\'est réalisable que si la borne basse de sa sensibilité atteint « plancher − marge ». Seuls les réalisables entrent au front de Pareto et peuvent devenir le meilleur.',
    impact: 'Plus haut : moins d\'individus réalisables ; le front et la meilleure sensibilité apparaissent plus tard. Trop haut : aucun réalisable, et la campagne s\'arrête par la règle de stagnation sans résultat (simulation : 0,85 → premier réalisable à la génération 4 ; 0,88 → aucun, arrêt à la génération 15). Plus bas : plus d\'individus admis, au risque de retenir des compromis trop peu sensibles.',
    boucleReelle: 'Même règle ; la borne basse vient d\'un bootstrap des patientes sur les sorties hors pli. C\'est aussi la règle de promotion d\'un modèle.',
  },
  {
    cle: 'projet.marge_points', section: 'projet', libelle: 'Marge (points)', plage: [1, 5], unite: 'points',
    role: 'Marge de non-infériorité, en points de sensibilité, retranchée du plancher : seuil appliqué = plancher − marge ÷ 100 (0,85 et 3 points → 0,82). Elle absorbe l\'incertitude de la mesure.',
    impact: 'Plus petite : contrainte plus stricte, comme un plancher plus haut (simulation : 1 point → premier réalisable à la génération 15 au lieu de 4, à la limite de l\'arrêt par stagnation). Plus grande : plus permissive (5 points → réalisables dès la génération 1).',
  },
  {
    cle: 'projet.taux_rappel_cible', section: 'projet', libelle: 'Taux de rappel cible', plage: [0.02, 0.1],
    role: 'Part des examens que le programme de dépistage rappelle (0,05 = 5 %). La sensibilité est mesurée à ce taux : à faible prévalence, il fixe presque la spécificité, et les modèles se comparent à taux de rappel égal.',
    impact: 'Il reste le même pendant toute la campagne : c\'est le point de mesure, pas un résultat (le journal rappelle « mesurée au taux de rappel cible fixé à 5,0 % »). Simulation : sans effet sur les valeurs.',
    boucleReelle: 'Plus haut, la sensibilité est plus facile à atteindre, au prix de plus de faux positifs. Il doit être celui du programme visé, sinon les résultats ne sont pas transposables.',
  },
  {
    cle: 'evolution.population', section: 'evolution', libelle: 'Population', plage: [20, 500], unite: 'individus',
    role: 'Nombre d\'individus (configurations CephaloMesh) évalués à chaque génération.',
    impact: 'Le coût est proportionnel : 100 individus ≈ 0,19 cœur-heure par génération, 500 ≈ 0,97. Plus grand : recherche plus large, mais le budget s\'épuise plus tôt (500 avec 50 cœur-heures : arrêt par le budget à la génération 52). Plus petit : moins cher, progression plus bruitée (20 → premier réalisable à la génération 8). Le volume stocké croît aussi : une ligne par individu et par génération.',
    boucleReelle: 'La durée d\'une génération croît comme la population ; dans la simulation, elle ne dépend que des lots.',
  },
  {
    cle: 'evolution.generations_max', section: 'evolution', libelle: 'Générations au plus', plage: [10, 2000], unite: 'générations',
    role: 'Plafond dur du nombre de générations. La campagne s\'arrête souvent avant, par stagnation ou par le budget.',
    impact: 'Sert aux estimations « au plus » de durée et de coût. Plus haut : laisse la place de progresser tant que E-val bouge ; ne coûte rien si un autre arrêt survient avant. Plus bas : borne la durée, au risque de couper une progression.',
  },
  {
    cle: 'evolution.arret_stagnation', section: 'evolution', libelle: 'Arrêt si E-val stable (générations)', plage: [3, 100], unite: 'générations',
    role: 'Nombre de générations sans nouveau meilleur sur E-val (gain d\'au moins 0,1 point) après lequel la campagne s\'arrête. Le compteur tourne aussi tant qu\'aucun individu n\'est réalisable. Le journal prévient 3 générations avant.',
    impact: 'Plus petit : arrêt rapide, au risque de s\'arrêter sur un palier (simulation : 5 → arrêt à la génération 9, meilleur 0,837). Plus grand : laisse le temps de franchir les paliers, pour un coût plus élevé (40 → arrêt à la génération 181, meilleur 0,923, contre 89 et 0,913 par défaut). Doit dépasser la génération où apparaissent les premiers réalisables.',
    boucleReelle: 'E-val ne sert jamais à optimiser : seulement à arrêter la boucle et à choisir le front exporté.',
  },
  {
    cle: 'evolution.budget_coeur_heures', section: 'evolution', libelle: 'Budget (cœur-heures)', plage: [0.1, 100000], unite: 'cœur-heures',
    role: 'Plafond dur de calcul. Chaque génération consomme population × 20 000 examens × 0,35 ms ; la campagne s\'arrête dès que le budget est atteint.',
    impact: 'Plus bas : arrêt plus tôt (simulation : 5 cœur-heures → arrêt à la génération 26, meilleur 0,873). Les contrôles signalent quand le budget arrêtera la campagne avant le plafond de générations. Pendant le suivi, « Budget + » l\'augmente ; il ne se baisse jamais.',
  },
  {
    cle: 'evolution.mutations_mu', section: 'evolution', libelle: 'μ mutations', plage: [0, 4],
    role: 'Règle le nombre de mutations de chaque enfant : m = 1 + Poisson(μ), plafonné par « Mutations au plus ». Avec μ = 1, un peu moins de 2 mutations en moyenne.',
    impact: 'Simulation : change seulement le nombre de mutations listées par individu (journal, fiche), pas les sensibilités.',
    boucleReelle: 'Plus grand : exploration, par grands pas (plus de nouveautés, plus d\'enfants dégradés). Plus petit : affinage autour des meilleurs.',
  },
  {
    cle: 'evolution.mutations_max', section: 'evolution', libelle: 'Mutations au plus', plage: [1, 10],
    role: 'Plafond du nombre de mutations par enfant. À 1, seules des mutations uniques (affinage).',
    impact: 'Simulation : borne le nombre de mutations listées, sans effet sur les sensibilités.',
    boucleReelle: 'Plus haut : autorise les macro-mutations qui sortent d\'un palier. Plus bas : recherche plus prudente.',
  },
  {
    cle: 'evolution.iles', section: 'evolution', libelle: 'Îles', plage: [1, 8], unite: 'îles',
    role: 'Nombre de populations qui évoluent en parallèle avec des réglages différents ; les meilleurs individus migrent d\'une île à l\'autre, ce qui limite la convergence prématurée.',
    impact: 'Simulation : sans effet.',
    boucleReelle: 'Plus d\'îles : plus de diversité, au prix d\'une coordination entre populations. 1 : une seule population.',
  },
  {
    cle: 'evolution.graine', section: 'evolution', libelle: 'Graine', format: 'nombre entier',
    role: 'Point de départ du tirage aléatoire. Avec le numéro de campagne et de génération, elle fixe tout le tirage : une génération recalculée (reprise après panne ou pause) donne exactement le même résultat.',
    impact: 'Une autre graine donne un autre tirage pour les mêmes réglages (simulation : 42 → arrêt à la génération 89, meilleur 0,913 ; 7 → génération 100, 0,915). Utile pour vérifier qu\'un écart entre deux réglages n\'est pas dû au hasard. Deux campagnes de même graine diffèrent quand même : le numéro de campagne entre dans le tirage.',
  },
  {
    cle: 'simulation.duree_lot_ms', section: 'simulation', libelle: 'Durée d\'un lot (ms)', plage: [10, 60000], unite: 'ms',
    role: 'Temps simulé d\'évaluation d\'un lot d\'individus, en attendant le vrai moteur. Le suivi reçoit une progression à chaque lot.',
    impact: 'Aucun effet sur les résultats, seulement sur le rythme : une génération dure « durée d\'un lot × lots par génération » (300 ms × 10 = 3 s). À baisser pour une démonstration rapide, à monter pour observer le suivi en direct.',
  },
  {
    cle: 'simulation.lots_par_generation', section: 'simulation', libelle: 'Lots par génération', plage: [1, 100], unite: 'lots',
    role: 'Nombre d\'étapes de progression d\'une génération (« lot 3 sur 10 » dans le bandeau de suivi).',
    impact: 'Aucun effet sur les résultats. Plus de lots : progression plus fine et plus de messages ; la consommation du budget est répartie par lot.',
  },
]

export const champParCle = Object.fromEntries(champs.map(c => [c.cle, c])) as Record<string, ChampGuide>

// Nom de contrôle renvoyé par l'API (« plancher_sensibilite ») → réglage du guide.
export function champDuControle(nom: string): ChampGuide | undefined {
  return champs.find(c => c.cle.endsWith('.' + nom))
}

export function lire(cfg: CampaignConfig, cle: string): unknown {
  return cle.split('.').reduce<unknown>((o, k) => (o as Record<string, unknown> | undefined)?.[k], cfg)
}

export function ecrire(cfg: CampaignConfig, cle: string, valeur: unknown) {
  const parts = cle.split('.')
  const parent = parts.slice(0, -1).reduce<Record<string, unknown>>((o, k) => o[k] as Record<string, unknown>, cfg as unknown as Record<string, unknown>)
  parent[parts.at(-1)!] = valeur
}

// ---------- Effets calculés, mêmes formules que l'API Go ----------

/** Cœur-heures d'une génération : population × 20 000 examens × 0,35 ms (campaign.CostPerGeneration). */
export function coutGeneration(cfg: CampaignConfig): number {
  return (cfg.evolution.population * 20000 * 0.35e-3) / 3600
}

/** Nombre moyen de mutations par enfant : E[min(1 + Poisson(μ), max)]. */
export function mutationsMoyennes(mu: number, max: number): number {
  if (max <= 1) return 1
  let p = Math.exp(-mu) // P(Poisson = 0)
  let somme = 0
  let cumul = 0
  for (let k = 0; k < max - 1; k++) {
    somme += (1 + k) * p
    cumul += p
    p *= mu / (k + 1)
  }
  return somme + max * (1 - cumul)
}

export interface Effets {
  seuil: number // plancher − marge
  dureeGenerationS: number
  dureeMaxMin: number
  coutGeneration: number
  coutMax: number
  arretBudget: number | null // génération où le budget arrête la campagne, si avant le plafond
  mutations: number
  premiersRealisables: number | null // simulation : génération au plus tôt ; null = jamais
}

/**
 * Premiers individus réalisables dans la simulation, au plus tôt : le plafond simulé monte de 0,80 vers 0,90
 * (0,80 + 0,10 (1 − e^(−g/35))), un individu le dépasse d'au plus 0,035 et sa borne basse est 0,022 plus bas.
 */
export function premiersRealisables(seuil: number, plafond = 2000): number | null {
  const besoin = (seuil - 0.813) / 0.1 // part du progrès simulé nécessaire
  if (besoin <= 0) return 1
  if (besoin >= 1) return null
  const g = Math.ceil(-35 * Math.log(1 - besoin))
  return g > plafond ? null : Math.max(1, g)
}

export function effets(cfg: CampaignConfig): Effets {
  const seuil = cfg.projet.plancher_sensibilite - cfg.projet.marge_points / 100
  const dureeGenerationS = (cfg.simulation.duree_lot_ms * cfg.simulation.lots_par_generation) / 1000
  const cg = coutGeneration(cfg)
  const gens = cfg.evolution.generations_max
  // Dernière génération jouée : l'orchestrateur en démarre une tant que le budget n'est pas atteint.
  const arret = cg > 0 ? Math.ceil(cfg.evolution.budget_coeur_heures / cg - 1e-9) : null
  return {
    seuil,
    dureeGenerationS,
    dureeMaxMin: (dureeGenerationS * gens) / 60,
    coutGeneration: cg,
    coutMax: cg * gens,
    arretBudget: arret !== null && arret < gens ? arret : null,
    mutations: mutationsMoyennes(cfg.evolution.mutations_mu, cfg.evolution.mutations_max),
    premiersRealisables: premiersRealisables(seuil, gens),
  }
}
