// Suivi des entraînements du CNN (/suivi/entrainements) : types renvoyés par l'API et aides d'affichage.

export interface EpoqueEntrainement { epoque: number; perte: number; auc_image: number | null; auc_sein: number | null; duree_s: number }

export interface ResumeEntrainement {
  nom: string
  statut: 'en_cours' | 'termine' | 'echec' | 'interrompu' | 'sans_nouvelles' | 'en_attente_gpu'
  phase: string
  etape?: string
  debut?: string
  maj?: string
  epoque: number
  epoques?: number
  meilleure_auc_sein: number | null
  auc_sein_validation: number | null
  message?: string
  avec_etat: boolean
  duree_epoque_s?: number | null
  pas?: number
  pas_par_epoque?: number
  debut_epoque?: string
  debut_phase?: string
  patience?: number
  avec_parametres?: boolean
  demande?: { action: 'relancer' | 'supprimer'; par: string; date: string }
}

export interface EtatEntrainement {
  pas?: number
  pas_par_epoque?: number
  pretraites?: number
  a_pretraiter?: number
  perte_courante?: number
  plan?: string[]
  appareil?: string
  precision?: string
  lot?: number
  accumulation?: number
  taille_entree?: number[]
  commande?: string
  donnees?: Record<string, { images?: number; seins_malins?: number; patientes?: number; seins?: number }>
}

export interface MesuresEntrainement {
  images: number
  seins: number
  seins_malins: number
  auc_image: number | null
  auc_sein: number | null
  seuil?: number
  sensibilite_sein?: number | null
  specificite_sein?: number | null
}

export interface RapportEntrainement {
  tache?: string
  modele?: { architecture: string; preentraine_imagenet: boolean; taille_entree: number[]; epoque_retenue: number }
  ajustement?: MesuresEntrainement
  validation?: MesuresEntrainement
}

export interface DetailEntrainement extends ResumeEntrainement {
  etat: EtatEntrainement | null
  journal: EpoqueEntrainement[]
  rapport: RapportEntrainement | null
}

export const statutEntrainement: Record<string, { libelle: string; classe: string }> = {
  en_cours: { libelle: 'En cours', classe: 'en_cours' },
  termine: { libelle: 'Terminé', classe: 'terminee' },
  echec: { libelle: 'En échec', classe: 'en_echec' },
  interrompu: { libelle: 'Interrompu', classe: 'arretee' },
  sans_nouvelles: { libelle: 'Sans nouvelles', classe: 'en_pause' },
  en_attente_gpu: { libelle: 'En attente du GPU', classe: 'en_pause' },
}

/** Finalisé : terminé, en échec ou interrompu. « Sans nouvelles » et « En attente du GPU » restent avec les en cours : ils reprendront. */
export function estFinalise(statut: string): boolean {
  return statut === 'termine' || statut === 'echec' || statut === 'interrompu'
}

export type VueEntrainements = 'en-cours' | 'finalises'

/** Répartit la liste (déjà triée, plus récent en tête) entre les deux onglets. */
export function repartir<T extends { statut: string }>(liste: T[]): Record<VueEntrainements, T[]> {
  return { 'en-cours': liste.filter(e => !estFinalise(e.statut)), 'finalises': liste.filter(e => estFinalise(e.statut)) }
}

export const phaseEntrainement: Record<string, string> = {
  preparation: 'Préparation',
  index: 'Index des images',
  pretraitement: 'Prétraitement des images',
  calibrage: 'Mesure du lot sur le GPU',
  entrainement: 'Entraînement',
  evaluation: 'Évaluation finale',
  termine: 'Terminé',
}

/** Avancement global entre 0 et 1, ou null s'il ne peut pas être estimé. */
export function avancement(d: Pick<DetailEntrainement, 'statut' | 'phase' | 'epoque' | 'epoques' | 'etat'>): number | null {
  if (d.statut === 'termine') return 1
  if (d.phase === 'pretraitement' && d.etat?.a_pretraiter) return (d.etat.pretraites ?? 0) / d.etat.a_pretraiter
  if (!d.epoques) return null
  const pas = d.etat?.pas_par_epoque ? Math.min(1, (d.etat.pas ?? 0) / d.etat.pas_par_epoque) : 0
  // L'époque en cours est celle d'après la dernière terminée, sauf si etat.json la donne déjà avec son pas.
  const faites = d.etat?.pas_par_epoque && (d.etat.pas ?? 0) > 0 ? d.epoque - 1 + pas : d.epoque
  return Math.max(0, Math.min(1, faites / d.epoques))
}

/** Durée moyenne d'une époque, en secondes (journal), pour estimer la fin. */
export function dureeMoyenneEpoque(journal: EpoqueEntrainement[]): number | null {
  const d = journal.map(e => e.duree_s).filter(x => x > 0)
  return d.length ? d.reduce((a, b) => a + b, 0) / d.length : null
}

export function duree(secondes: number | null | undefined): string {
  if (secondes === null || secondes === undefined || !Number.isFinite(secondes)) return '—'
  const s = Math.round(secondes)
  if (s < 60) return `${s} s`
  const m = Math.round(s / 60)
  if (m < 60) return `${m} min`
  return `${Math.floor(m / 60)} h ${String(m % 60).padStart(2, '0')}`
}

// ---------- Relancer, supprimer ----------

/** Libellé et explication du bouton « Relancer » selon ce que le veilleur fera. */
export function relance(e: Pick<ResumeEntrainement, 'statut' | 'avec_parametres'>): { possible: boolean; libelle: string; explication: string } {
  if (e.statut === 'en_cours') return { possible: false, libelle: 'Relancer', explication: 'Il tourne déjà.' }
  if (e.avec_parametres && e.statut !== 'termine') {
    return { possible: true, libelle: 'Reprendre', explication: 'Il reprendra à sa dernière époque finie, avec les mêmes réglages, dès que le GPU sera libre.' }
  }
  return {
    possible: true, libelle: 'Relancer',
    explication: e.avec_parametres
      ? 'Un nouvel entraînement, avec les mêmes réglages, partira dès que le GPU sera libre. Celui-ci reste dans les finalisés.'
      : 'Cet entraînement ne peut pas reprendre où il s\'est arrêté (lancé avant les points de reprise) : un nouvel entraînement, avec les mêmes réglages, partira de zéro dès que le GPU sera libre.',
  }
}

export const libelleDemande: Record<string, string> = {
  relancer: 'Relance demandée',
  supprimer: 'Suppression demandée',
}

// ---------- Estimation des temps de fin ----------

/** Relevé pris par la page à chaque rafraîchissement : sert à mesurer la vitesse réelle. */
export interface Releve { t: number; phase: string; epoque: number; pas: number; pretraites: number }

export interface Vitesse { pas: number | null; images: number | null }

const FENETRE_MS = 3 * 60_000
const ECART_MIN_MS = 15_000

/** Vitesse observée entre le plus ancien et le plus récent relevé des 3 dernières minutes, dans la même phase et la même époque. */
export function vitesseObservee(releves: Releve[]): Vitesse {
  const dernier = releves.at(-1)
  if (!dernier) return { pas: null, images: null }
  const fenetre = releves.filter(r => dernier.t - r.t <= FENETRE_MS && r.phase === dernier.phase && r.epoque === dernier.epoque)
  const premier = fenetre[0]!
  const dt = (dernier.t - premier.t) / 1000
  if (dt * 1000 < ECART_MIN_MS) return { pas: null, images: null }
  const v = (a: number, b: number) => (b > a ? (b - a) / dt : null)
  return { pas: v(premier.pas, dernier.pas), images: v(premier.pretraites, dernier.pretraites) }
}

export interface Estimation {
  /** Durée d'une époque complète (apprentissage + mesure en ajustement), en secondes. */
  dureeEpoque: number | null
  /** Numéro de l'époque en cours et temps restant pour la finir. */
  epoqueCourante: number | null
  resteEpoque: number | null
  /** Fin si toutes les époques prévues sont faites, évaluation finale comprise. */
  resteMax: number | null
  /** Fin au plus tôt si l'arrêt anticipé se déclenche (null s'il ne peut pas avancer la fin). */
  resteMin: number | null
  epoqueArret: number | null
  restePretraitement: number | null
}

type Entree = ResumeEntrainement & { journal?: EpoqueEntrainement[]; etat?: EtatEntrainement | null }

const vide: Estimation = { dureeEpoque: null, epoqueCourante: null, resteEpoque: null, resteMax: null, resteMin: null, epoqueArret: null, restePretraitement: null }

/**
 * Temps restants, en secondes. Vitesse d'apprentissage, par ordre de préférence : relevés de la page, pas faits
 * depuis le début de l'époque (etat.json), durée des époques passées. La mesure en ajustement est comptée à
 * 35 % du coût d'une image d'apprentissage (passage avant seul), l'évaluation finale à deux mesures.
 */
export function estimer(d: Entree, maintenant: number, vitesse: Vitesse = { pas: null, images: null }): Estimation {
  if (d.statut !== 'en_cours') return vide
  const ms = (s?: string) => (s ? new Date(s).getTime() : NaN)
  const ecoule = Number.isFinite(ms(d.maj)) ? Math.max(0, (maintenant - ms(d.maj)) / 1000) : 0
  const est: Estimation = { ...vide }

  if (d.phase === 'pretraitement' && d.etat?.a_pretraiter) {
    const fait = d.etat.pretraites ?? 0
    const depuis = (ms(d.maj) - ms(d.debut_phase)) / 1000
    const v = vitesse.images ?? (fait > 0 && depuis > 0 ? fait / depuis : null)
    if (v) est.restePretraitement = Math.max(0, (d.etat.a_pretraiter - fait) / v - ecoule)
    return est
  }
  if (d.phase !== 'entrainement' || !d.epoques) return est

  const n = d.pas_par_epoque ?? d.etat?.pas_par_epoque ?? 0
  const dn = d.etat?.donnees
  const facteur = 1 + 0.35 * ((dn?.ajustement?.images ?? 0) / (dn?.entrainement?.images || 1) || 0.2)
  const dTyp = d.duree_epoque_s ?? null

  // Époque en cours et pas faits.
  let c: number, p: number | null
  if (d.avec_etat) {
    const finie = d.etape === 'époque terminée'
    c = finie ? d.epoque + 1 : Math.max(1, d.epoque)
    p = finie ? 0 : (d.pas ?? 0)
  } else {
    c = (d.journal?.length ?? d.epoque) + 1
    p = null
  }
  if (c > d.epoques) c = d.epoques

  const depuisEpoque = (ms(d.maj) - ms(d.debut_epoque)) / 1000
  const v = vitesse.pas ?? (p && depuisEpoque > 0 ? p / depuisEpoque : null) ?? (dTyp && n ? (n * facteur) / dTyp : null)
  const apprentissage = v && n ? n / v : dTyp ? dTyp / facteur : null
  const mesure = apprentissage !== null ? apprentissage * (facteur - 1) : null
  est.dureeEpoque = dTyp ?? (apprentissage !== null && mesure !== null ? apprentissage + mesure : null)
  est.epoqueCourante = c

  if (p !== null && v && n && mesure !== null) {
    const base = d.etape === 'mesure en ajustement' ? mesure : (n - Math.min(p, n)) / v + mesure
    est.resteEpoque = Math.max(0, base - ecoule)
  } else if (dTyp) {
    est.resteEpoque = Math.max(0, dTyp - ecoule)
  }
  if (est.resteEpoque === null || est.dureeEpoque === null) return est

  const finale = 2 * (mesure ?? 0)
  const apres = (derniere: number) => est.resteEpoque! + Math.max(0, derniere - c) * est.dureeEpoque! + finale
  est.resteMax = apres(d.epoques)

  // Arrêt anticipé : pas de progrès de l'AUC par sein pendant « patience » époques après la meilleure.
  const j = d.journal ?? []
  if (j.length) {
    let meilleure = 0
    let auc = -1
    for (const e of j) if ((e.auc_sein ?? -1) > auc) { auc = e.auc_sein ?? -1; meilleure = e.epoque }
    const arret = Math.max(c, Math.min(d.epoques, meilleure + (d.patience || 5)))
    if (arret < d.epoques) {
      est.epoqueArret = arret
      est.resteMin = apres(arret)
    }
  }
  return est
}

/** « 16:40 », ou « mer. 16:40 » si ce n'est pas aujourd'hui. */
export function heureFin(maintenant: number, resteS: number): string {
  const fin = new Date(maintenant + resteS * 1000)
  const h = fin.toLocaleTimeString('fr-BE', { hour: '2-digit', minute: '2-digit' })
  return fin.toDateString() === new Date(maintenant).toDateString() ? h : `${fin.toLocaleDateString('fr-BE', { weekday: 'short' })} ${h}`
}
