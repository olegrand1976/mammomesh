import { beforeEach, describe, expect, it, vi } from 'vitest'
import { mockNuxtImport } from '@nuxt/test-utils/runtime'
import { createPinia, setActivePinia } from 'pinia'
import type { Campaign, CampaignEvent, Generation } from '~/types'

// API simulée : chaque route renvoie l'état courant de « server ».
const server = vi.hoisted(() => ({
  campaign: {} as Partial<Campaign>,
  lastEventId: 10,
  generations: [] as Partial<Generation>[],
  calls: [] as { path: string; query?: Record<string, unknown> }[],
}))
mockNuxtImport('useApi', () => () => ({
  api: async (path: string, opts: { query?: Record<string, unknown> } = {}) => {
    server.calls.push({ path, query: opts.query })
    if (path.endsWith('/generations')) {
      const after = Number(opts.query?.after ?? 0)
      return { generations: server.generations.filter(g => (g.number ?? 0) > after) }
    }
    if (path.endsWith('/front')) return { generation: server.generations.length, front: [], previous: [] }
    if (path.endsWith('/events')) return { events: [] }
    if (path.endsWith('/control')) return { campaign: { ...server.campaign, state: 'en_pause' } }
    return { campaign: server.campaign, last_event_id: server.lastEventId }
  },
}))

// EventSource simulé : le test envoie lui-même les messages.
class FakeSource {
  static last: FakeSource
  listeners: Record<string, ((m: MessageEvent) => void)[]> = {}
  onopen: (() => void) | null = null
  onerror: (() => void) | null = null
  closed = false
  constructor(public url: string) { FakeSource.last = this }
  addEventListener(type: string, fn: (m: MessageEvent) => void) { (this.listeners[type] ??= []).push(fn) }
  emit(type: string, data: unknown) { for (const fn of this.listeners[type] ?? []) fn({ data: JSON.stringify(data) } as MessageEvent) }
  close() { this.closed = true }
}
vi.stubGlobal('EventSource', FakeSource)

const ev = (id: number, type: string): CampaignEvent =>
  ({ id, campaign_id: 1, type, message: type, payload: {}, created_at: '2026-10-26T10:00:00Z' })
const flush = () => new Promise(r => setTimeout(r, 0))

describe('store campagne', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    server.campaign = { id: 1, state: 'en_cours', generation: 1 }
    server.lastEventId = 10
    server.generations = [{ number: 1 }]
    server.calls = []
  })

  it('reprend le flux là où la lecture REST s\'arrête', async () => {
    const store = useCampagneStore()
    await store.load(1)
    expect(FakeSource.last.url).toBe('/api/campaigns/1/stream?lastEventId=10')
    FakeSource.last.onopen?.()
    expect(store.connected).toBe(true)
  })

  it('ignore un événement déjà connu et ajoute les nouveaux en tête', async () => {
    const store = useCampagneStore()
    await store.load(1)
    FakeSource.last.emit('nouveau_meilleur', ev(9, 'nouveau_meilleur'))
    FakeSource.last.emit('nouveau_meilleur', ev(11, 'nouveau_meilleur'))
    FakeSource.last.emit('individus_ecartes', ev(12, 'individus_ecartes'))
    FakeSource.last.emit('individus_ecartes', ev(12, 'individus_ecartes'))
    await flush()
    expect(store.journal.map(e => e.id)).toEqual([12, 11])
  })

  it('ne relit que les nouvelles générations', async () => {
    const store = useCampagneStore()
    await store.load(1)
    server.generations = [{ number: 1 }, { number: 2 }]
    server.campaign = { ...server.campaign, generation: 2 }
    FakeSource.last.emit('generation_terminee', ev(11, 'generation_terminee'))
    await flush(); await flush()
    expect(store.generations.map(g => g.number)).toEqual([1, 2])
    expect(server.calls.some(c => c.path.endsWith('/generations') && c.query?.after === 1)).toBe(true)
    expect(store.campaign?.generation).toBe(2)
  })

  it('efface la progression quand la campagne quitte l\'état en cours', async () => {
    const store = useCampagneStore()
    await store.load(1)
    FakeSource.last.emit('progression', { campaign_id: 1, generation: 2, lot: 1, lots: 4, budget_used: 1, budget_total: 10 })
    expect(store.progress?.lot).toBe(1)
    server.campaign = { ...server.campaign, state: 'terminee' }
    FakeSource.last.emit('campagne_terminee', ev(11, 'campagne_terminee'))
    await flush(); await flush()
    expect(store.progress).toBeNull()
    expect(store.campaign?.state).toBe('terminee')
  })

  it('recharge tout sur « resync » et ferme l\'ancien flux', async () => {
    const store = useCampagneStore()
    await store.load(1)
    const first = FakeSource.last
    server.lastEventId = 40
    first.emit('resync', {})
    await flush(); await flush()
    expect(first.closed).toBe(true)
    expect(FakeSource.last.url).toContain('lastEventId=40')
  })

  it('applique la réponse d\'un réglage', async () => {
    const store = useCampagneStore()
    await store.load(1)
    await store.control('pause')
    expect(store.campaign?.state).toBe('en_pause')
  })
})
