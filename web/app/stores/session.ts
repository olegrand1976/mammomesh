import { defineStore } from 'pinia'
import type { Permissions, User } from '~/types'

export const useSessionStore = defineStore('session', () => {
  const user = ref<User | null>(null)
  const permissions = ref<Permissions | null>(null)
  const { api } = useApi()

  async function refresh(): Promise<boolean> {
    try {
      const res = await api<{ user: User; permissions: Permissions }>('/me')
      user.value = res.user
      permissions.value = res.permissions
      return true
    } catch {
      user.value = null
      permissions.value = null
      return false
    }
  }

  async function login(username: string, password: string) {
    await api('/auth/login', { method: 'POST', body: { username, password } })
    await refresh()
  }

  async function changePassword(actuel: string, nouveau: string) {
    await api('/auth/password', { method: 'POST', body: { actuel, nouveau } })
    await refresh()
  }

  async function logout() {
    await api('/auth/logout', { method: 'POST' }).catch(() => {})
    user.value = null
    permissions.value = null
    await navigateTo('/login')
  }

  return { user, permissions, refresh, login, changePassword, logout }
})
