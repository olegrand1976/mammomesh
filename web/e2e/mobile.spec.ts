import { expect, test } from '@playwright/test'
import { ADMIN, connexion } from './aide'

// Écran de téléphone : aucune barre de défilement horizontale ; le menu est un tiroir.
test('suivi lisible sur téléphone @mobile', async ({ page }) => {
  await connexion(page, ADMIN.nouveau)
  await expect(page).toHaveURL(/\/suivi$/)
  const debordement = () => page.evaluate(() => document.documentElement.scrollWidth - window.innerWidth)
  expect(await debordement()).toBeLessThanOrEqual(0)
  // Barre compacte en haut ; le menu reste fermé tant qu'on ne l'ouvre pas.
  expect((await page.locator('header.barre-mobile').boundingBox())!.height).toBeLessThan(60)
  await expect(page.getByRole('link', { name: 'Campagnes', exact: true })).toBeHidden()
  await page.getByRole('button', { name: 'Menu' }).click()
  await page.getByRole('link', { name: 'Campagnes', exact: true }).click()
  await expect(page).toHaveURL(/\/suivi\/campagnes$/)
  await expect(page.getByRole('button', { name: 'Fermer le menu' })).toBeHidden()   // fermé après la navigation
  await page.getByRole('link', { name: 'Essai e2e', exact: true }).click()
  await expect(page.locator('canvas').first()).toBeVisible()
  expect(await debordement()).toBeLessThanOrEqual(0)
})
