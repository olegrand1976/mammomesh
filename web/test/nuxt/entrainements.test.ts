import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { describe, expect, it } from 'vitest'
import { avancement, duree, dureeMoyenneEpoque, estFinalise, phaseEntrainement, repartir, statutEntrainement } from '~/utils/entrainements'

const racine = resolve(__dirname, '../..')

describe('suivi des entraînements', () => {
  it('avancement : époques terminées, pas dans l\'époque, prétraitement, fin', () => {
    expect(avancement({ statut: 'en_cours', phase: 'calibrage', epoque: 0, epoques: undefined, etat: null })).toBeNull()
    expect(avancement({ statut: 'en_cours', phase: 'entrainement', epoque: 3, epoques: 15, etat: null })).toBeCloseTo(0.2)
    expect(avancement({ statut: 'en_cours', phase: 'entrainement', epoque: 4, epoques: 10, etat: { pas: 50, pas_par_epoque: 100 } })).toBeCloseTo(0.35)
    expect(avancement({ statut: 'en_cours', phase: 'entrainement', epoque: 4, epoques: 10, etat: { pas: 100, pas_par_epoque: 100 } })).toBeCloseTo(0.4)
    expect(avancement({ statut: 'en_cours', phase: 'entrainement', epoque: 0, epoques: 10, etat: { pas: 0, pas_par_epoque: 100 } })).toBe(0)
    expect(avancement({ statut: 'en_cours', phase: 'pretraitement', epoque: 0, epoques: undefined, etat: { pretraites: 250, a_pretraiter: 1000 } })).toBeCloseTo(0.25)
    expect(avancement({ statut: 'termine', phase: 'termine', epoque: 9, epoques: 15, etat: null })).toBe(1)
  })

  it('onglets : en cours (y compris sans nouvelles) et finalisés', () => {
    const l = [{ nom: 'a', statut: 'en_cours' }, { nom: 'b', statut: 'termine' }, { nom: 'c', statut: 'sans_nouvelles' }, { nom: 'd', statut: 'echec' }, { nom: 'e', statut: 'interrompu' }]
    const r = repartir(l)
    expect(r['en-cours'].map(e => e.nom)).toEqual(['a', 'c'])
    expect(r.finalises.map(e => e.nom)).toEqual(['b', 'd', 'e'])
    for (const s of Object.keys(statutEntrainement)) expect(typeof estFinalise(s)).toBe('boolean')
  })

  it('durées', () => {
    expect(dureeMoyenneEpoque([{ epoque: 1, perte: 0.6, auc_image: 0.7, auc_sein: 0.7, duree_s: 600 }, { epoque: 2, perte: 0.5, auc_image: 0.7, auc_sein: 0.7, duree_s: 400 }])).toBe(500)
    expect(dureeMoyenneEpoque([])).toBeNull()
    expect(duree(42)).toBe('42 s')
    expect(duree(25 * 60)).toBe('25 min')
    expect(duree(3 * 3600 + 5 * 60)).toBe('3 h 05')
    expect(duree(null)).toBe('—')
  })

  it('chaque statut et chaque phase écrits par le moteur et l\'API ont un libellé', () => {
    const go = readFileSync(resolve(racine, '../backend/internal/entrainements/entrainements.go'), 'utf8')
    for (const [, s] of go.matchAll(/(?:EnCours|Termine|Echec|Interrompu|SansNouvelle)\s+= "([a-z_]+)"/g)) expect(statutEntrainement[s!], s).toBeDefined()
    const py = ['suivi.py', 'entrainement.py', '__main__.py'].map(f => readFileSync(resolve(racine, `../engine/mammomesh_engine/cnn/${f}`), 'utf8')).join('\n')
    const phases = [...py.matchAll(/phase\("([a-z_]+)"/g), ...py.matchAll(/"phase": "([a-z_]+)"/g)].map(m => m[1]!)
    expect(phases.length).toBeGreaterThan(4)
    for (const ph of phases) expect(phaseEntrainement[ph], ph).toBeDefined()
    for (const st of [...py.matchAll(/fin\("([a-z_]+)"/g)].map(m => m[1]!)) expect(statutEntrainement[st], st).toBeDefined()
  })
})
