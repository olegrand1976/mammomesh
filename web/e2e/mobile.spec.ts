import { expect, test } from '@playwright/test'
import { ADMIN, connexion } from './aide'

// Écran de téléphone : aucune barre de défilement horizontale sur les pages de suivi.
test('suivi lisible sur téléphone @mobile', async ({ page }) => {
  await connexion(page, ADMIN.nouveau)
  await expect(page).toHaveURL(/\/suivi$/)
  const debordement = () => page.evaluate(() => document.documentElement.scrollWidth - window.innerWidth)
  expect(await debordement()).toBeLessThanOrEqual(0)
  // En-tête compact : deux lignes au plus (logo + déconnexion, navigation).
  expect((await page.locator('header.app').boundingBox())!.height).toBeLessThan(110)
  await page.getByRole('link', { name: 'Essai e2e', exact: true }).click()
  await expect(page.locator('canvas').first()).toBeVisible()
  expect(await debordement()).toBeLessThanOrEqual(0)
})
