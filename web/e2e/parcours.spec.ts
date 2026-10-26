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
  await expect(page.getByRole('navigation', { name: 'Navigation du suivi' })).toHaveCount(0)

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
  await page.getByLabel('Population', { exact: true }).fill('30')
  await page.getByLabel('Générations au plus', { exact: true }).fill('12')
  await page.getByLabel('Durée d\'un lot (ms)', { exact: true }).fill('80')
  await page.getByLabel('Lots par génération', { exact: true }).fill('3')
  await page.getByRole('button', { name: 'Vérifier la configuration' }).click()
  await expect(page.getByRole('heading', { name: 'Contrôles avant lancement' })).toBeVisible()
  await page.getByRole('button', { name: 'Lancer la campagne' }).click()

  await expect(page).toHaveURL(/\/suivi\/campagnes\/\d+$/)
  await expect(page.getByRole('heading', { name: 'Essai e2e' })).toBeVisible()
  await expect(page.getByText('En direct', { exact: true })).toBeVisible()
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

  // Aides de l'écran de suivi : une par zone, toutes ouvertes d'un clic ; un point du front ouvre sa fiche.
  await expect(page.locator('.aide-zone')).toHaveCount(0)
  await page.getByRole('button', { name: 'À quoi sert « Front de Pareto » ?' }).click()
  await expect(page.locator('#aide-front')).toContainText('non dominés')
  await page.getByRole('button', { name: 'Afficher toutes les aides' }).click()
  await expect(page.locator('.aide-zone')).toHaveCount(4) // bandeau, progrès, front, journal
  await page.getByRole('button', { name: 'Masquer les aides' }).click()
  await expect(page.locator('.aide-zone')).toHaveCount(0)
  await page.getByText(/^Liste des individus du front/).click()
  await page.getByRole('button', { name: 'Fiche' }).last().click()
  await expect(page.getByRole('heading', { name: /^Fiche / })).toBeInViewport()
  await expect(page.getByRole('rowheader', { name: 'Sensibilité (E-val)' })).toBeVisible()

  await page.getByRole('link', { name: 'Campagnes' }).first().click()
  await expect(page.getByRole('row', { name: /Essai e2e/ })).toContainText('Terminée')
  expect(erreurs).toEqual([])
})

test('aides de l\'écran Nouvelle campagne et effet des réglages', async ({ page }) => {
  await connexion(page, ADMIN.nouveau)
  await page.goto('/suivi/nouvelle')
  const aides = page.locator('.aide')
  await expect(aides).toHaveCount(16) // un par réglage
  await expect(aides.first()).toBeHidden()
  await page.getByRole('button', { name: 'Afficher toutes les aides' }).click()
  for (const a of await aides.all()) await expect(a).toBeVisible()
  await page.getByRole('button', { name: 'Masquer les aides' }).click()
  await expect(aides.first()).toBeHidden()

  // Plancher plus exigeant : alerte, puis levée en allongeant l'arrêt par stagnation.
  const panneau = page.getByRole('region', { name: 'Effet de vos réglages' })
  await expect(panneau).toContainText('0,82')
  await page.getByLabel('Plancher de sensibilité', { exact: true }).fill('0.88')
  await expect(panneau).toContainText('0,85')
  await expect(panneau.locator('.alertes')).toContainText('génération 17 au plus tôt')
  await page.getByLabel('Arrêt si E-val stable (générations)', { exact: true }).fill('40')
  await expect(panneau.locator('.alertes')).toHaveCount(0)
  await expect(page.locator('.champ.modifie')).toHaveCount(2)
  await page.locator('.champ.modifie').first().getByRole('button', { name: 'rétablir' }).click()
  await expect(page.getByLabel('Plancher de sensibilité', { exact: true })).toHaveValue('0.85')

  // Contrôles : libellés lisibles plutôt que les clés techniques.
  await page.getByRole('button', { name: 'Vérifier la configuration' }).click()
  await expect(page.getByRole('cell', { name: 'Plancher de sensibilité' })).toBeVisible()
  await expect(page.getByText('Réglages modifiés : evolution.arret_stagnation')).toBeVisible()
})

test('configuration par défaut lancée sans rien modifier, puis arrêtée', async ({ page }) => {
  await connexion(page, ADMIN.nouveau)
  await page.goto('/suivi/nouvelle')
  await expect(page.getByLabel('Nom', { exact: true })).toHaveValue(/.+/)
  await page.getByRole('button', { name: 'Vérifier la configuration' }).click()
  await expect(page.getByText('Réglages modifiés')).toHaveCount(0) // rien de modifié : pas de liste
  await page.getByRole('button', { name: 'Lancer la campagne' }).click()
  await expect(page).toHaveURL(/\/suivi\/campagnes\/\d+$/)
  await expect(page.getByText(/^[1-9]\d* \/ 200$/)).toBeVisible({ timeout: 15_000 }) // une génération dure ~3 s
  page.once('dialog', d => d.accept())
  await page.getByRole('button', { name: 'Arrêter' }).click()
  await expect(page.locator('.badge.arretee')).toBeVisible()
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
