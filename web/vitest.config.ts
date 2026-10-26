import { defineVitestConfig } from '@nuxt/test-utils/config'

// Tests unitaires dans un environnement Nuxt simulé (auto-imports, Pinia), sans serveur.
export default defineVitestConfig({
  test: {
    environment: 'nuxt',
    include: ['test/nuxt/**/*.test.ts'],
    environmentOptions: { nuxt: { domEnvironment: 'happy-dom' } },
  },
})
