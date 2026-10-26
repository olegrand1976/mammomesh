// Appels à l'API Go. Le cookie de session (HttpOnly, SameSite=Strict) part automatiquement.
export function useApi() {
  async function api<T>(path: string, opts: { method?: string; body?: unknown; query?: Record<string, unknown> } = {}): Promise<T> {
    try {
      return await $fetch<T>(`/api${path}`, {
        method: (opts.method ?? 'GET') as 'GET',
        body: opts.body as Record<string, unknown> | undefined,
        query: opts.query,
        credentials: 'same-origin',
      })
    } catch (err: unknown) {
      const e = err as { statusCode?: number; data?: { erreur?: string } }
      if (e.statusCode === 401 && path !== '/auth/login') {
        await navigateTo('/login')
      }
      throw new Error(e.data?.erreur ?? 'Erreur de communication avec le serveur')
    }
  }
  return { api }
}
