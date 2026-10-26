// Protège /suivi : l'API Go reste seule juge (route /api/me) ; Nuxt ne fait que rediriger.
export default defineNuxtRouteMiddleware(async () => {
  if (import.meta.server) return
  const session = useSessionStore()
  if (session.user) return
  if (!(await session.refresh())) {
    return navigateTo('/login')
  }
})
