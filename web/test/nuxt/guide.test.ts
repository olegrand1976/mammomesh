import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { describe, expect, it } from 'vitest'
import type { CampaignConfig } from '~/types'
import { champDuControle, champs, coutGeneration, effets, lire, mutationsMoyennes, premiersRealisables } from '~/utils/guideCampagne'

// Configuration par défaut de l'API (campaign.Default).
const defaut = (): CampaignConfig => ({
  projet: { nom: 'Campagne simulée', objectif: 'Valider la chaîne de suivi de bout en bout', lignee: 'ORION', plancher_sensibilite: 0.85, marge_points: 3, taux_rappel_cible: 0.05 },
  evolution: { population: 100, generations_max: 200, arret_stagnation: 15, budget_coeur_heures: 50, mutations_mu: 1, mutations_max: 5, iles: 1, graine: 42 },
  simulation: { duree_lot_ms: 300, lots_par_generation: 10 },
})
const feuilles = (o: object, pre = ''): string[] => Object.entries(o).flatMap(([k, v]) =>
  v && typeof v === 'object' ? feuilles(v, `${pre}${k}.`) : [`${pre}${k}`])
const racine = resolve(__dirname, '../..')

describe('guide de l\'écran Nouvelle campagne', () => {
  it('chaque réglage de la configuration a son aide, et seulement eux', () => {
    expect(champs.map(c => c.cle).sort()).toEqual(feuilles(defaut()).sort())
    for (const c of champs) expect(lire(defaut(), c.cle)).not.toBeUndefined()
  })

  it('chaque réglage a un champ dans l\'écran', () => {
    const page = readFileSync(resolve(racine, 'app/pages/suivi/nouvelle.vue'), 'utf8')
    for (const c of champs) expect(page, c.cle).toContain(`cle="${c.cle}"`)
  })

  it('le guide du dépôt reprend mot pour mot chaque aide', () => {
    const doc = readFileSync(resolve(racine, '../docs/guide-nouvelle-campagne.md'), 'utf8')
    for (const c of champs) {
      expect(doc, c.cle).toContain(`#### ${c.libelle}`)
      expect(doc, c.cle).toContain(c.role)
      expect(doc, c.cle).toContain(c.impact)
      if (c.boucleReelle) expect(doc, c.cle).toContain(c.boucleReelle)
    }
  })

  it('les noms de contrôles de l\'API trouvent leur libellé', () => {
    for (const nom of ['nom', 'lignee', 'plancher_sensibilite', 'population', 'iles', 'lots_par_generation']) {
      expect(champDuControle(nom)).toBeDefined()
    }
    expect(champDuControle('budget')).toBeUndefined()
  })
})

describe('effets calculés (mêmes formules que l\'API)', () => {
  it('coût d\'une génération : population × 20 000 × 0,35 ms', () => {
    expect(coutGeneration(defaut())).toBeCloseTo(0.19444, 4)
  })

  it('mutations moyennes : E[min(1 + Poisson(μ), max)]', () => {
    expect(mutationsMoyennes(1, 5)).toBeCloseTo(1.9957, 3) // « un peu moins de 2 »
    expect(mutationsMoyennes(0, 5)).toBe(1)
    expect(mutationsMoyennes(3, 1)).toBe(1)
  })

  it('premiers réalisables de la simulation, au plus tôt', () => {
    expect(premiersRealisables(0.82)).toBe(3) // 4 constaté
    expect(premiersRealisables(0.85)).toBe(17) // plancher 0,88 : après l'arrêt par stagnation (15)
    expect(premiersRealisables(0.95)).toBeNull()
  })

  it('défaut : 3 s et 0,19 cœur-h par génération, pas d\'arrêt par le budget', () => {
    const e = effets(defaut())
    expect(e.seuil).toBeCloseTo(0.82)
    expect(e.dureeGenerationS).toBe(3)
    expect(e.dureeMaxMin).toBe(10)
    expect(e.arretBudget).toBeNull()
  })

  it('budget limitant : 500 individus et 50 cœur-h → arrêt après la génération 52 (comme l\'API)', () => {
    const c = defaut()
    c.evolution.population = 500
    expect(effets(c).arretBudget).toBe(52)
  })
})

describe('guide de l\'écran de suivi', async () => {
  const { zones } = await import('~/utils/guideSuivi')
  const fichiers: Record<string, string> = {
    bandeau: 'app/components/StatusBanner.vue', progres: 'app/components/ProgressChart.vue', front: 'app/components/FrontChart.vue',
    fiche: 'app/components/IndividualCard.vue', journal: 'app/components/JournalList.vue',
  }

  it('chaque zone a son bouton d\'aide et son emplacement', () => {
    for (const z of zones) {
      const src = readFileSync(resolve(racine, fichiers[z.cle]!), 'utf8')
      expect(src, z.cle).toContain(`<AideZone zone="${z.cle}" />`)
      expect(src, z.cle).toContain(`id="aide-${z.cle}-ici"`)
    }
  })

  it('le guide du dépôt reprend mot pour mot chaque élément', () => {
    const doc = readFileSync(resolve(racine, '../docs/guide-suivi-campagne.md'), 'utf8')
    for (const z of zones) {
      expect(doc).toContain(`. ${z.titre}`)
      for (const e of z.elements) expect(doc, e.nom).toContain(`| ${e.nom} | ${e.description} |`)
      if (z.aSavoir) expect(doc).toContain(z.aSavoir)
    }
  })
})

describe('premiers réalisables après un départ', () => {
  it('soustrait l\'avancée acquise', () => {
    expect(premiersRealisables(0.85)).toBe(17)
    expect(premiersRealisables(0.85, 2000, 12)).toBe(5)
    expect(premiersRealisables(0.85, 2000, 40)).toBe(1)
  })
})
