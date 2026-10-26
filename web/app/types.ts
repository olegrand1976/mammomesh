// Types partagés, alignés sur les réponses JSON de l'API Go.

export interface User {
  id: number
  username: string
  display_name: string
  role: string
  must_change_password: boolean
}

export interface Permissions {
  lancer_campagnes: boolean
  proposer_config: boolean
  jetons_proposition: boolean
}

export interface CampaignConfig {
  projet: { nom: string; objectif: string; lignee: string; plancher_sensibilite: number; marge_points: number; taux_rappel_cible: number }
  evolution: {
    population: number; generations_max: number; arret_stagnation: number; budget_coeur_heures: number
    mutations_mu: number; mutations_max: number; iles: number; graine: number
  }
  simulation: { duree_lot_ms: number; lots_par_generation: number }
  depart?: Depart | null
}

/** Départ d'une campagne : front de la génération d'une campagne finie, parents de la génération 1. */
export interface Depart { campagne_id: number; generation: number }

export interface Campaign {
  id: number
  name: string
  lineage: string
  state: 'en_cours' | 'en_pause' | 'arretee' | 'terminee' | 'en_echec'
  generation: number
  max_generations: number
  budget_total: number
  budget_used: number
  best_sensitivity: number | null
  stagnation: number
  config_hash: string
  config: CampaignConfig
  created_at: string
  started_at: string | null
  finished_at: string | null
  updated_at: string
}

export interface Generation {
  number: number
  kind: string
  started_at: string
  finished_at: string
  population: number
  feasible: number
  best_sensitivity: number | null // null : aucun individu réalisable
  median_sensitivity: number
  best_cost_ms: number
  median_cost_ms: number
  eval_gap: number | null
}

export interface Individual {
  id: number
  campaign_id: number
  generation: number
  name: string
  parent_name: string | null
  mutations: string[]
  genome: Record<string, unknown>
  sensitivity: number
  sensitivity_low: number
  sensitivity_high: number
  sensitivity_eval: number
  cost_ms: number
  feasible: boolean
  on_front: boolean
}

export interface CampaignEvent {
  id: number
  campaign_id: number | null
  type: string
  message: string
  payload: Record<string, unknown>
  created_at: string
}

export interface Progress {
  campaign_id: number
  generation: number
  lot: number
  lots: number
  budget_used: number
  budget_total: number
}

export interface Check { nom: string; ok: boolean; detail: string }

export interface Draft {
  id: number
  config: CampaignConfig
  diff: { champ: string; avant: unknown; apres: unknown }[]
  verification: { ok: boolean; controles: Check[]; duree_estimee_min: number; coeur_heures_estimees: number } | null
  verified_ok: boolean
  launched_campaign_id: number | null
}

export interface MCPToken {
  id: number
  name: string
  scope: 'lecture' | 'proposition'
  created_at: string
  expires_at: string
  last_used_at: string | null
  revoked_at: string | null
}

export const stateLabel: Record<string, string> = {
  en_cours: 'En cours',
  en_pause: 'En pause',
  arretee: 'Arrêtée',
  terminee: 'Terminée',
  en_echec: 'En échec',
}

export const roleLabel: Record<string, string> = {
  admin: 'Administrateur',
  responsable_scientifique: 'Responsable scientifique',
  data_manager: 'Data manager',
  radiologue: 'Radiologue référent',
  lecteur: 'Lecteur',
}

export function fr(n: number | null | undefined, digits = 3): string {
  if (n === null || n === undefined) return '—'
  return n.toLocaleString('fr-BE', { minimumFractionDigits: digits, maximumFractionDigits: digits })
}
