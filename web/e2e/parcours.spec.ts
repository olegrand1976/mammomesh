import { expect, test } from '@playwright/test'
import { ADMIN, connexion } from './aide'

// Parcours complet d'un premier utilisateur, sur une base neuve.
test.describe.configure({ mode: 'serial' })

test('page de présentation et accès à la connexion', async ({ page }) => {
  await page.goto('/')
  await expect(page.getByRole('heading', { level: 1 })).toBeVisible()
  await page.getByRole('link', { name: 'Se connecter' }).first().click()
  await expect(page).toHaveURL(/\/login$/)
})

test('mauvais mot de passe refusé', async ({ page }) => {
  await connexion(page, 'mauvais-mot-de-passe', { echec: true })
  await expect(page.getByRole('alert')).toContainText('identifiant ou mot de passe incorrect')
  await expect(page).toHaveURL(/\/login$/)
})

test('premier accès : changement imposé du mot de passe par défaut', async ({ page }) => {
  await connexion(page, ADMIN.defaut)
  await expect(page).toHaveURL(/\/suivi\/compte$/)
  await expect(page.getByRole('status')).toContainText('mot de passe par défaut')

  await page.goto('/suivi')
  await expect(page).toHaveURL(/\/suivi\/compte$/) // le suivi reste fermé

  const enregistrer = page.getByRole('button', { name: 'Enregistrer' })
  await page.getByLabel('Mot de passe actuel').fill(ADMIN.defaut)
  await page.getByLabel(/Nouveau mot de passe/).fill('court')
  await expect(page.getByText(/Encore \d+ caractère/)).toBeVisible()
  await expect(enregistrer).toBeDisabled()

  await page.getByLabel(/Nouveau mot de passe/).fill(ADMIN.nouveau)
  await page.getByLabel('Confirmation').fill(ADMIN.nouveau + 'x')
  await expect(page.getByText('Les deux saisies diffèrent.')).toBeVisible()
  await page.getByLabel('Confirmation').fill(ADMIN.nouveau)
  await enregistrer.click()

  await expect(page).toHaveURL(/\/suivi$/)
  await expect(page.getByRole('heading', { name: 'Campagnes' })).toBeVisible()
})

test('campagne lancée depuis l\'interface et suivie en direct', async ({ page }) => {
  await connexion(page, ADMIN.nouveau)
  await expect(page).toHaveURL(/\/suivi$/)
  // Erreurs du navigateur relevées une fois connecté (avant, /login sonde la session : 401 attendu).
  const erreurs: string[] = []
  page.on('console', m => { if (m.type() === 'error') erreurs.push(m.text()) })
  page.on('pageerror', e => erreurs.push(e.message))
  await page.getByRole('link', { name: 'Nouvelle campagne' }).first().click()
  await page.getByLabel('Nom', { exact: true }).fill('Essai e2e')
  await page.getByLabel('Population').fill('30')
  await page.getByLabel('Générations au plus').fill('12')
  await page.getByLabel('Durée d\'un lot (ms)').fill('80')
  await page.getByLabel('Lots par génération').fill('3')
  await page.getByRole('button', { name: 'Vérifier la configuration' }).click()
  await expect(page.getByRole('heading', { name: 'Contrôles avant lancement' })).toBeVisible()
  await page.getByRole('button', { name: 'Lancer la campagne' }).click()

  await expect(page).toHaveURL(/\/suivi\/campagnes\/\d+$/)
  await expect(page.getByRole('heading', { name: 'Essai e2e' })).toBeVisible()
  await expect(page.getByText('En direct')).toBeVisible()
  // Le compteur avance sans recharger la page, jusqu'à la fin.
  await expect(page.getByText(/^[1-9]\d* \/ 12$/)).toBeVisible({ timeout: 20_000 })
  await expect(page.locator('.badge.terminee')).toBeVisible({ timeout: 40_000 })
  await expect(page.getByText('12 / 12')).toBeVisible()
  await expect(page.locator('canvas')).toHaveCount(2)
  await expect(page.getByRole('heading', { name: 'Journal' })).toBeVisible()
  const journal = page.locator('ol.journal li')
  await expect(journal.locator('xpath=.').and(page.locator('.generation_terminee'))).toHaveCount(12)
  await page.getByLabel('Filtrer le journal').selectOption('nouveau_meilleur')
  await expect(journal.first()).toHaveClass(/nouveau_meilleur/)
  expect(await journal.evaluateAll(li => li.every(e => e.classList.contains('nouveau_meilleur')))).toBe(true)

  await page.getByRole('link', { name: 'Campagnes' }).first().click()
  await expect(page.getByRole('row', { name: /Essai e2e/ })).toContainText('Terminée')
  expect(erreurs).toEqual([])
})

test('jeton MCP créé, affiché une fois, puis révoqué', async ({ page }) => {
  await connexion(page, ADMIN.nouveau)
  await page.goto('/suivi/jetons')
  await page.getByLabel('Nom du jeton').fill('jeton e2e')
  await page.getByRole('button', { name: 'Créer' }).click()
  await expect(page.locator('code.token').first()).toHaveText(/^mmk_/)
  const ligne = page.getByRole('row', { name: /jeton e2e/ })
  await expect(ligne).toContainText('actif')
  page.once('dialog', d => d.accept())
  await ligne.getByRole('button', { name: 'Révoquer' }).click()
  await expect(ligne).toContainText('révoqué')
})

test('déconnexion : le suivi redemande la connexion', async ({ page }) => {
  await connexion(page, ADMIN.nouveau)
  await expect(page).toHaveURL(/\/suivi$/)
  await page.getByRole('button', { name: 'Déconnexion' }).click()
  await expect(page).toHaveURL(/\/login$/)
  await page.goto('/suivi')
  await expect(page).toHaveURL(/\/login$/)
})
