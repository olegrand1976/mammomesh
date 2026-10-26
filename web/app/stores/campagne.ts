import { defineStore } from 'pinia'
import type { Campaign, CampaignEvent, Generation, Individual, Progress } from '~/types'

// Types d'événements du journal diffusés par le flux SSE (un écouteur par type).
const JOURNAL_TYPES = [
  'campagne_lancee', 'generation_terminee', 'nouveau_meilleur', 'individus_ecartes', 'eval_stable',
  'etat_change', 'budget_augmente', 'reprise_apres_redemarrage', 'campagne_terminee', 'campagne_en_echec',
]
const STATE_TYPES = new Set(['etat_change', 'budget_augmente', 'campagne_terminee', 'campagne_en_echec', 'reprise_apres_redemarrage'])

/**
 * Un seul flux par onglet, partagé par tous les panneaux.
 * L'état initial vient de l'API REST ; ensuite seuls les changements arrivent.
 */
export const useCampagneStore = defineStore('campagne', () => {
  const { api } = useApi()

  const campaign = ref<Campaign | null>(null)
  const generations = ref<Generation[]>([])
  const front = ref<Individual[]>([])
  const previousFront = ref<Individual[]>([])
  const frontGeneration = ref(0)
  const journal = ref<CampaignEvent[]>([])
  const progress = ref<Progress | null>(null)
  const connected = ref(false)
  const lastUpdate = ref<Date | null>(null)

  let source: EventSource | null = null
  let currentId = 0
  let lastEventId = 0

  async function load(id: number) {
    disconnect()
    currentId = id
    const [c, g, f, e] = await Promise.all([
      api<{ campaign: Campaign; last_event_id: number }>(`/campaigns/${id}`),
      api<{ generations: Generation[] }>(`/campaigns/${id}/generations`),
      api<{ generation: number; front: Individual[]; previous?: Individual[] }>(`/campaigns/${id}/front`),
      api<{ events: CampaignEvent[] }>(`/campaigns/${id}/events`, { query: { limit: 200 } }),
    ])
    campaign.value = c.campaign
    lastEventId = c.last_event_id
    generations.value = g.generations
    front.value = f.front
    previousFront.value = f.previous ?? []
    frontGeneration.value = f.generation
    journal.value = e.events
    progress.value = null
    lastUpdate.value = new Date()
    connect()
  }

  function connect() {
    // lastEventId : pas de trou entre la lecture REST et l'ouverture du flux.
    // Après une coupure, le navigateur renvoie lui-même Last-Event-ID.
    source = new EventSource(`/api/campaigns/${currentId}/stream?lastEventId=${lastEventId}`, { withCredentials: true })
    source.onopen = () => { connected.value = true }
    source.onerror = () => { connected.value = false }
    source.addEventListener('progression', (m) => {
      progress.value = JSON.parse((m as MessageEvent).data) as Progress
      lastUpdate.value = new Date()
    })
    source.addEventListener('resync', () => { void load(currentId) })
    for (const type of JOURNAL_TYPES) {
      source.addEventListener(type, (m) => onJournal(JSON.parse((m as MessageEvent).data) as CampaignEvent))
    }
  }

  async function onJournal(e: CampaignEvent) {
    if (e.id <= lastEventId) return
    lastEventId = e.id
    journal.value = [e, ...journal.value].slice(0, 500)
    lastUpdate.value = new Date()
    if (e.type === 'generation_terminee') {
      await onGeneration()
    }
    if (STATE_TYPES.has(e.type) || e.type === 'generation_terminee') {
      const c = await api<{ campaign: Campaign }>(`/campaigns/${currentId}`)
      campaign.value = c.campaign
      if (c.campaign.state !== 'en_cours') progress.value = null
    }
  }

  async function onGeneration() {
    const after = generations.value.at(-1)?.number ?? 0
    const [g, f] = await Promise.all([
      api<{ generations: Generation[] }>(`/campaigns/${currentId}/generations`, { query: { after } }),
      api<{ generation: number; front: Individual[]; previous?: Individual[] }>(`/campaigns/${currentId}/front`),
    ])
    generations.value = [...generations.value, ...g.generations]
    previousFront.value = f.previous ?? front.value
    front.value = f.front
    frontGeneration.value = f.generation
  }

  async function control(action: 'pause' | 'reprise' | 'arret' | 'budget', hausse = 0) {
    const res = await api<{ campaign: Campaign }>(`/campaigns/${currentId}/control`, {
      method: 'POST', body: { action, hausse_budget: hausse },
    })
    campaign.value = res.campaign
  }

  function disconnect() {
    source?.close()
    source = null
    connected.value = false
  }

  return { campaign, generations, front, previousFront, frontGeneration, journal, progress, connected, lastUpdate, load, control, disconnect }
})
