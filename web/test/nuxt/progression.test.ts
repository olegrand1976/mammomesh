import { describe, expect, it } from 'vitest'
import { exemples, lirePerte, lireProgression, syntheseEntrainement } from '~/utils/progression'

describe('lecture de l\'évolution d\'un entraînement', () => {
  it('chaque exemple du guide reçoit la lecture annoncée', () => {
    for (const e of exemples) expect(lireProgression(e.journal)!.verdict, e.titre).toBe(e.attendu)
  })
  it('surapprentissage : l\'époque gardée est le record, pas la dernière', () => {
    const lu = lireProgression(exemples[1]!.journal)!
    expect(lu.epoqueMeilleure).toBe(4)
    expect(lu.phrases.join(' ')).toContain('époque 4 qui sera gardée')
  })
  it('une seule époque : trop tôt pour juger ; journal vide : rien', () => {
    expect(lireProgression([{ epoque: 1, perte: 0.6, auc_image: 0.6, auc_sein: 0.62, duree_s: 10 }])!.verdict).toBe('debut')
    expect(lireProgression([])).toBeNull()
  })
  it('synthèse finale en phrases simples', () => {
    const j = exemples[0]!.journal
    const s = syntheseEntrainement(j, { modele: { architecture: 'x', preentraine_imagenet: true, taille_entree: [], epoque_retenue: 8 },
      validation: { images: 60, seins: 30, seins_malins: 8, auc_image: 0.7, auc_sein: 0.77, seuil: 0.4, sensibilite_sein: 0.85, specificite_sein: 0.6 } })
    const t = s.join(' ')
    expect(t).toContain('85 seins malins signalés sur 100')
    expect(t).toContain('40 fausses alertes')
    expect(t).toContain('30 seins seulement')
  })
})

describe('lecture de l\'évolution d\'une campagne', async () => {
  const { exemplesCampagne, generationsExemple, lireCampagne, raisonArret, syntheseCampagne, exempleFront } = await import('~/utils/progressionCampagne')
  it('chaque exemple du guide reçoit la lecture annoncée', () => {
    for (const e of exemplesCampagne) expect(lireCampagne(generationsExemple(e), 0.85, 15)!.verdict, e.titre).toBe(e.attendu)
  })
  it('sans version réalisable : le dit, avec le minimum en clair', () => {
    const g = generationsExemple({ titre: '', attendu: 'sans_realisable', points: [[1, null], [2, null], [3, null]], lecture: '' })
    const lu = lireCampagne(g, 0.85)!
    expect(lu.verdict).toBe('sans_realisable')
    expect(lu.phrases[0]).toContain('85 cancers sur 100')
  })
  it('l\'exemple de front est cohérent : seule D est dominée', () => {
    for (const v of exempleFront) {
      const domine = exempleFront.some(w => w !== v && w.sensibilite >= v.sensibilite && w.cout <= v.cout)
      expect(!domine, v.nom).toBe(v.front)
    }
  })
  it('synthèse d\'une campagne finie par stagnation', () => {
    const g = generationsExemple(exemplesCampagne[0]!)
    const c = { id: 1, name: 'X', lineage: 'ORION', state: 'terminee', generation: 45, max_generations: 200, budget_total: 50, budget_used: 12.3,
      best_sensitivity: 0.877, stagnation: 15, config_hash: '', config: { projet: { plancher_sensibilite: 0.85 }, evolution: { arret_stagnation: 15 } },
      created_at: '', started_at: '2026-10-27T10:00:00Z', finished_at: '2026-10-27T12:30:00Z', updated_at: '' } as never
    expect(raisonArret(c)).toContain('plus aucun progrès')
    const front = [{ name: 'A', feasible: true, sensitivity_eval: 0.86, cost_ms: 2.1 }, { name: 'C', feasible: true, sensitivity_eval: 0.9, cost_ms: 6 }] as never
    const t = syntheseCampagne(c, g, front).join(' ')
    expect(t).toContain('après 45 générations en 2,5 h')
    expect(t).toContain('de la plus rapide, A')
  })
})

describe('lecture de la perte', () => {
  it('explique la perte et chiffre son évolution', () => {
    const j = [{ epoque: 1, perte: 0.42, auc_image: 0.62, auc_sein: 0.639, duree_s: 10 }, { epoque: 2, perte: 0.413, auc_image: 0.657, auc_sein: 0.677, duree_s: 10 }]
    const t = lirePerte(j).join(' ')
    expect(t).toContain('images d\'apprentissage')
    expect(t).toContain('de 0,420 à 0,413 (−1,7 %)')
    expect(lireProgression(j)!.phrases.join(' ')).toContain('axe de droite est zoomé')
    expect(lirePerte([])).toEqual([])
  })
})
