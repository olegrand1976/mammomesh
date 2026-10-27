// Suivi des entraînements du CNN (/suivi/entrainements) : types renvoyés par l'API et aides d'affichage.

export interface EpoqueEntrainement { epoque: number; perte: number; auc_image: number | null; auc_sein: number | null; duree_s: number }

export interface ResumeEntrainement {
  nom: string
  statut: 'en_cours' | 'termine' | 'echec' | 'interrompu' | 'sans_nouvelles'
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
}

/** Finalisé : terminé, en échec ou interrompu. « Sans nouvelles » reste avec les en cours : il peut reprendre. */
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
