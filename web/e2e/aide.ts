import type { Page } from '@playwright/test'

// Compte créé par « migrate » sur une base neuve ; le parcours change son mot de passe.
export const ADMIN = { user: 'admin', defaut: 'admin$1', nouveau: 'mot-de-passe-e2e-solide' }

// Attend d'avoir quitté /login, sauf si l'échec est attendu.
export async function connexion(page: Page, motDePasse: string, { echec = false } = {}) {
  await page.goto('/login')
  await page.getByLabel('Identifiant').fill(ADMIN.user)
  await page.getByLabel('Mot de passe').fill(motDePasse)
  await page.getByRole('button', { name: 'Se connecter' }).click()
  if (!echec) await page.waitForURL(u => u.pathname.startsWith('/suivi'))
}
