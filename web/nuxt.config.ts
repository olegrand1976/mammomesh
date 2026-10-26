// Interface MammoMesh : page de présentation pré-rendue, login et interface de suivi (rendu côté navigateur).
// En production, Caddy sert /api et /mcp vers l'API Go ; en développement, le proxy de Nuxt s'en charge.
const apiTarget = process.env.API_URL ?? 'http://localhost:8080'

export default defineNuxtConfig({
  compatibilityDate: '2026-10-01',
  modules: ['@pinia/nuxt'],
  css: ['~/assets/css/main.css'],
  devtools: { enabled: false },
  app: {
    head: {
      htmlAttrs: { lang: 'fr' },
      title: 'MammoMesh',
      meta: [
        { name: 'viewport', content: 'width=device-width, initial-scale=1' },
        { name: 'description', content: 'MammoMesh : recherche autonome d\'un couple CNN + CephaloMesh pour la mammographie. Outil de recherche, sans usage clinique.' },
      ],
      link: [{ rel: 'icon', type: 'image/svg+xml', href: '/favicon.svg' }],
    },
  },
  routeRules: {
    '/': { prerender: true },
    '/login': { ssr: false },
    '/suivi/**': { ssr: false },
  },
  nitro: {
    devProxy: {
      '/api': { target: `${apiTarget}/api`, changeOrigin: true },
    },
  },
  typescript: { strict: true },
})
