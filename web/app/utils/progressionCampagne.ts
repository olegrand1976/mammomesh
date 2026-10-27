// Lecture, en mots simples, de l'évolution d'une campagne (écran /suivi/campagnes/:id) et synthèse finale.
// Les exemples du guide sont vérifiés par un test (même lecture que sur une vraie campagne).
import type { Campaign, Generation, Individual } from '~/types'

export type VerdictCampagne = 'debut' | 'sans_realisable' | 'progresse' | 'plateau'

export const titresCampagne: Record<VerdictCampagne, string> = {
  debut: 'Trop tôt pour juger',
  sans_realisable: 'Aucune version assez sûre pour l\'instant',
  progresse: 'La recherche progresse',
  plateau: 'Plus de progrès pour l\'instant',
}

const fr = (x: number, d = 3) => x.toLocaleString('fr-BE', { minimumFractionDigits: d, maximumFractionDigits: d })
const pts = (x: number) => (x * 100).toLocaleString('fr-BE', { maximumFractionDigits: 1 })
const sur100 = (s: number) => Math.round(s * 100)
const gens = (n: number) => `${n} génération${n > 1 ? 's' : ''}`
const ms = (x: number) => `${x.toLocaleString('fr-BE', { maximumFractionDigits: 1 })} ms`

export interface LectureCampagne { verdict: VerdictCampagne; titre: string; phrases: string[] }

/** Lecture de la courbe des générations. `arret` : générations sans progrès avant l'arrêt automatique. */
export function lireCampagne(generations: Generation[], plancher: number, arret = 15): LectureCampagne | null {
  if (!generations.length) return null
  const phrases: string[] = []
  const derniere = generations[generations.length - 1]!
  const realisables = generations.filter(g => g.best_sensitivity !== null)
  let verdict: VerdictCampagne
  if (!realisables.length) {
    verdict = generations.length < 3 ? 'debut' : 'sans_realisable'
    phrases.push(`Après ${gens(generations.length)}, aucune version ne trouve encore assez de cancers : le minimum exigé est ${fr(plancher, 2)}, soit ${sur100(plancher)} cancers sur 100. La campagne continue de chercher ; c'est fréquent au début.`)
    return { verdict, titre: titresCampagne[verdict], phrases }
  }
  const premiere = realisables[0]!
  let record = premiere
  for (const g of realisables) if (g.best_sensitivity! > record.best_sensitivity! + 1e-9) record = g
  const depuis = derniere.number - record.number
  const gain = record.best_sensitivity! - premiere.best_sensitivity!
  verdict = generations.length < 3 ? 'debut' : depuis >= Math.max(3, Math.ceil(arret / 3)) ? 'plateau' : 'progresse'

  phrases.push(`Meilleure sensibilité sur les images de contrôle (E-val) : ${fr(premiere.best_sensitivity!)} à la génération ${premiere.number} (première version assez sûre), ${fr(record.best_sensitivity!)} à la génération ${record.number}, soit +${pts(gain)} points. Concrètement : sur 100 cancers, ${sur100(record.best_sensitivity!)} trouvés au lieu de ${sur100(premiere.best_sensitivity!)}.`)
  phrases.push(`Dernière génération (${derniere.number}) : ${derniere.feasible} version${derniere.feasible > 1 ? 's' : ''} sur ${derniere.population} atteignent le minimum de ${sur100(plancher)} cancers sur 100 ; la moitié de la population en trouve au moins ${sur100(derniere.median_sensitivity)}.`)
  if (premiere.best_cost_ms > 0 && derniere.best_cost_ms > 0) {
    const r = derniere.best_cost_ms / premiere.best_cost_ms
    phrases.push(Math.abs(r - 1) < 0.05
      ? `La version la plus rapide met toujours environ ${ms(derniere.best_cost_ms)} par examen.`
      : `La version la plus rapide met ${ms(derniere.best_cost_ms)} par examen, contre ${ms(premiere.best_cost_ms)} au départ (${r < 1 ? 'plus rapide' : 'plus lente'}).`)
  }
  if (verdict === 'progresse' && depuis === 0) phrases.push('Nouveau record à la dernière génération : la recherche trouve encore mieux.')
  else if (verdict === 'progresse') phrases.push(`Dernier record il y a ${gens(depuis)} ; la recherche continue normalement.`)
  if (verdict === 'plateau') {
    const reste = arret - depuis
    phrases.push(`Pas de nouveau record depuis ${gens(depuis)}. ${reste > 0 ? `Si cela dure encore ${gens(reste)}, la campagne s'arrêtera d'elle-même : continuer n'apporterait plus rien.` : 'La campagne va s\'arrêter d\'elle-même.'}`)
  }
  if (derniere.eval_gap !== null && derniere.eval_gap > 0.03) phrases.push(`Les versions retenues font ${pts(derniere.eval_gap)} points de mieux sur les images d'entraînement de la recherche (E-ajust) que sur les images de contrôle (E-val) : elles se sont un peu trop adaptées aux premières. Seule la valeur E-val compte pour juger.`)
  return { verdict, titre: titresCampagne[verdict], phrases }
}

/** Pourquoi la campagne s'est arrêtée, en mots simples. */
export function raisonArret(c: Campaign): string {
  if (c.state === 'arretee') return 'arrêtée par une personne'
  if (c.state === 'en_echec') return 'interrompue par une erreur (voir le journal)'
  if (c.generation >= c.max_generations) return `nombre maximal de générations atteint (${c.max_generations})`
  if (c.budget_used >= c.budget_total - 1e-9) return 'temps de calcul alloué épuisé'
  return `plus aucun progrès pendant ${gens(c.config?.evolution?.arret_stagnation ?? 15)} (arrêt automatique)`
}

/** Synthèse globale d'une campagne finie (terminée, arrêtée ou en échec). */
export function syntheseCampagne(c: Campaign, generations: Generation[], front: Individual[]): string[] {
  const out: string[] = []
  const debut = c.started_at ? new Date(c.started_at).getTime() : null
  const fin = c.finished_at ? new Date(c.finished_at).getTime() : null
  const duree = debut && fin ? Math.max(0, (fin - debut) / 60000) : null
  const dureeTxt = duree === null ? '' : duree < 90 ? ` en ${Math.round(duree)} min` : ` en ${(duree / 60).toLocaleString('fr-BE', { maximumFractionDigits: 1 })} h`
  out.push(`Campagne finie après ${gens(c.generation)}${dureeTxt} : ${raisonArret(c)}. Temps de calcul utilisé : ${c.budget_used.toLocaleString('fr-BE', { maximumFractionDigits: 1 })} sur ${c.budget_total.toLocaleString('fr-BE', { maximumFractionDigits: 1 })} cœur-heures.`)
  const lu = lireCampagne(generations, c.config?.projet?.plancher_sensibilite ?? 0.85, c.config?.evolution?.arret_stagnation ?? 15)
  if (lu && lu.verdict !== 'sans_realisable' && lu.verdict !== 'debut') out.push(lu.phrases[0]!)
  else out.push('Aucune version n\'a atteint le minimum de cancers trouvés : la campagne ne propose aucun candidat. Il faut revoir les réglages ou attendre un meilleur CNN.')
  const f = front.filter(i => i.feasible).sort((a, b) => a.cost_ms - b.cost_ms)
  if (f.length) {
    const rapide = f[0]!, sur = f.reduce((a, b) => (b.sensitivity_eval > a.sensitivity_eval ? b : a))
    out.push(f.length === 1
      ? `Une seule version retenue au final : ${rapide.name} (${sur100(rapide.sensitivity_eval)} cancers sur 100 en contrôle, ${ms(rapide.cost_ms)} par examen).`
      : `${f.length} versions retenues au final, toutes des compromis différents : de la plus rapide, ${rapide.name} (${sur100(rapide.sensitivity_eval)} cancers sur 100 en contrôle, ${ms(rapide.cost_ms)} par examen), à la plus sûre, ${sur.name} (${sur100(sur.sensitivity_eval)} sur 100, ${ms(sur.cost_ms)}).`)
    out.push('Suite : une personne habilitée choisit la version à retenir selon le besoin (plus de cancers trouvés ou plus de rapidité), ou lance une nouvelle campagne qui repart de celle-ci.')
  }
  return out
}

/** Scénarios types, chiffres illustratifs : meilleure sensibilité (E-val) par génération ; null = aucune version assez sûre. */
export interface ExempleCampagne { titre: string; attendu: VerdictCampagne; points: [number, number | null][]; lecture: string }

export const exemplesCampagne: ExempleCampagne[] = [
  {
    titre: 'Une recherche qui progresse puis se stabilise',
    attendu: 'plateau',
    points: [[1, 0.851], [5, 0.858], [10, 0.866], [15, 0.871], [20, 0.874], [25, 0.876], [30, 0.877], [35, 0.877], [40, 0.877], [45, 0.877]],
    lecture: 'Progrès rapides au début (+1,5 point en 10 générations), puis de plus en plus lents. Après la génération 30, plus aucun record : au bout de 15 générations sans progrès, la campagne s\'arrête d\'elle-même, car continuer ne servirait plus à rien. Sur 100 cancers, la meilleure version en trouve 88 au lieu de 85.',
  },
  {
    titre: 'Aucune version assez sûre au départ',
    attendu: 'progresse',
    points: [[1, null], [2, null], [3, null], [4, null], [5, 0.844], [6, 0.849], [7, 0.853], [8, 0.858]],
    lecture: 'Pendant 4 générations, aucune version ne trouve assez de cancers : le graphique reste vide, c\'est normal. À la génération 5, une première version franchit le minimum, puis les suivantes l\'améliorent à chaque génération.',
  },
]

/** Exemple de front de Pareto : quatre versions, une seule n'est pas un bon compromis. */
export const exempleFront: { nom: string; sensibilite: number; cout: number; front: boolean; pourquoi: string }[] = [
  { nom: 'A', sensibilite: 0.86, cout: 2.1, front: true, pourquoi: 'La plus rapide.' },
  { nom: 'B', sensibilite: 0.88, cout: 3.4, front: true, pourquoi: 'Plus sûre que A, un peu plus lente : un autre compromis.' },
  { nom: 'C', sensibilite: 0.90, cout: 6.0, front: true, pourquoi: 'La plus sûre, mais la plus lente.' },
  { nom: 'D', sensibilite: 0.87, cout: 4.0, front: false, pourquoi: 'Écartée : B fait mieux sur les deux points (plus sûre et plus rapide).' },
]

/** Transforme un exemple en générations, pour la même lecture qu'une vraie campagne. */
export function generationsExemple(e: ExempleCampagne): Generation[] {
  return e.points.map(([n, s]) => ({ number: n, kind: 'normale', started_at: '', finished_at: '', population: 100, feasible: s === null ? 0 : 12,
    best_sensitivity: s, median_sensitivity: s === null ? 0.79 : s - 0.04, best_cost_ms: 3, median_cost_ms: 5, eval_gap: 0.01 }))
}
