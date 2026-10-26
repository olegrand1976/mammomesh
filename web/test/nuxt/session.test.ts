import { beforeEach, describe, expect, it, vi } from 'vitest'
import { mockNuxtImport } from '@nuxt/test-utils/runtime'
import { createPinia, setActivePinia } from 'pinia'

const state = vi.hoisted(() => ({ mustChange: true, calls: [] as { path: string; body?: unknown }[] }))
mockNuxtImport('useApi', () => () => ({
  api: async (path: string, opts: { body?: unknown } = {}) => {
    state.calls.push({ path, body: opts.body })
    if (path === '/auth/password') state.mustChange = false
    if (path === '/me') {
      return {
        user: { id: 1, username: 'admin', display_name: '', role: 'admin', must_change_password: state.mustChange },
        permissions: { lancer_campagnes: true, proposer_config: true, jetons_proposition: true },
      }
    }
    return {}
  },
}))
const navigate = vi.hoisted(() => vi.fn())
mockNuxtImport('navigateTo', () => navigate)

describe('session', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    state.mustChange = true
    state.calls = []
    navigate.mockReset()
  })

  it('connexion puis profil, avec le drapeau du mot de passe par défaut', async () => {
    const s = useSessionStore()
    await s.login('admin', 'admin$1')
    expect(state.calls.map(c => c.path)).toEqual(['/auth/login', '/me'])
    expect(s.user?.must_change_password).toBe(true)
  })

  it('le changement de mot de passe envoie les deux saisies et relit le profil', async () => {
    const s = useSessionStore()
    await s.refresh()
    await s.changePassword('admin$1', 'un-mot-de-passe-solide')
    expect(state.calls.find(c => c.path === '/auth/password')?.body).toEqual({ actuel: 'admin$1', nouveau: 'un-mot-de-passe-solide' })
    expect(s.user?.must_change_password).toBe(false)
  })

  it('la déconnexion vide la session et renvoie vers /login', async () => {
    const s = useSessionStore()
    await s.refresh()
    await s.logout()
    expect(s.user).toBeNull()
    expect(navigate).toHaveBeenCalledWith('/login')
  })
})
