// Protège /suivi : l'API Go reste seule juge (route /api/me) ; Nuxt ne fait que rediriger.
// Mot de passe par défaut : seule la page « Mon compte » reste accessible tant qu'il n'est pas changé.
export default defineNuxtRouteMiddleware(async (to) => {
  if (import.meta.server) return
  const session = useSessionStore()
  if (!session.user && !(await session.refresh())) {
    return navigateTo('/login')
  }
  if (session.user?.must_change_password && to.path !== '/suivi/compte') {
    return navigateTo('/suivi/compte')
  }
})
