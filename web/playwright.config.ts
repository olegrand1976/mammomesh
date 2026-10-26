import { defineConfig, devices } from '@playwright/test'

// Tests de bout en bout dans un navigateur : pile complète démarrée par Playwright
// (API + orchestrateur Go, Nuxt compilé, proxy à la place de Caddy), base dédiée remise à zéro.
//   E2E_DATABASE_URL  base jetable (obligatoire) ; E2E_REDIS_URL facultatif (vide : compteurs en mémoire)
//   MAMMOMESH_BIN     binaire Go (défaut ../backend/bin/mammomesh) ; `make e2e` construit tout.
const bin = process.env.MAMMOMESH_BIN ?? '../backend/bin/mammomesh'
const db = process.env.E2E_DATABASE_URL ?? ''
const base = 'http://127.0.0.1:14000'

export default defineConfig({
  testDir: './e2e',
  fullyParallel: false,
  workers: 1, // un seul parcours : le compte par défaut change de mot de passe une fois
  retries: process.env.CI ? 1 : 0,
  timeout: 60_000,
  reporter: process.env.CI ? [['list'], ['html', { open: 'never' }]] : 'list',
  use: {
    baseURL: base,
    locale: 'fr-BE',
    trace: 'retain-on-failure',
    screenshot: 'only-on-failure',
    // Navigateur déjà installé ailleurs (machine sans accès au dépôt de Playwright).
    launchOptions: process.env.PW_CHROMIUM_PATH ? { executablePath: process.env.PW_CHROMIUM_PATH } : {},
  },
  projects: [
    { name: 'bureau', use: { ...devices['Desktop Chrome'] }, grepInvert: /@mobile/ },
    // Après « bureau » : le mot de passe par défaut y a été changé.
    { name: 'mobile', use: { ...devices['Pixel 7'] }, grep: /@mobile/, dependencies: ['bureau'] },
  ],
  webServer: [
    {
      // Base remise à zéro, schéma et compte admin par défaut, puis orchestrateur et API.
      command: `node e2e/reset-db.mjs && ${bin} migrate && (${bin} orchestrator &) && exec ${bin} api`,
      url: 'http://127.0.0.1:18080/api/health',
      env: {
        ...process.env as Record<string, string>,
        DATABASE_URL: db, REDIS_URL: process.env.E2E_REDIS_URL ?? '', MCP_TOKEN_PEPPER: 'e2e',
        COOKIE_SECURE: 'false', LISTEN_ADDR: '127.0.0.1:18080', METRICS_ADDR: '127.0.0.1:19090',
        ADMIN_USERNAME: 'admin', ADMIN_PASSWORD: '', COALESCE_INTERVAL: '300ms', LOG_LEVEL: 'warn',
        IMAGES_URL: '', IMAGES_DIR: 'test-results/images-e2e',
      },
      reuseExistingServer: false,
      timeout: 60_000,
    },
    { command: 'node .output/server/index.mjs', url: 'http://127.0.0.1:13000/', env: { PORT: '13000', HOST: '127.0.0.1' }, reuseExistingServer: false },
    { command: 'node e2e/proxy.mjs', url: `${base}/api/health`, reuseExistingServer: false },
  ],
})
