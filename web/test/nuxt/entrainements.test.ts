import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { describe, expect, it } from 'vitest'
import { aideReglages, pausePossible, avancement, duree, dureeMoyenneEpoque, estFinalise, estimer, heureFin, libelleDemande, phaseEntrainement, relance, repartir, statutEntrainement, vitesseObservee, type ResumeEntrainement } from '~/utils/entrainements'

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

  describe('temps de fin', () => {
    const T = Date.parse('2026-10-27T13:00:00Z')
    const base: ResumeEntrainement = { nom: 'x', statut: 'en_cours', phase: 'entrainement', epoque: 3, epoques: 15, meilleure_auc_sein: 0.8,
      auc_sein_validation: null, avec_etat: true, maj: '2026-10-27T13:00:00Z', patience: 5 }
    const etat = { donnees: { entrainement: { images: 2000 }, ajustement: { images: 400 } } }   // mesure = 7 % de l'apprentissage
    const journal = [1, 2, 3].map(e => ({ epoque: e, perte: 0.6, auc_image: 0.7, auc_sein: e === 2 ? 0.8 : 0.7, duree_s: 1070 }))

    it('époque en cours d\'après les pas faits depuis son début, puis fin au plus tard et arrêt anticipé', () => {
      // Époque 4 : 100 pas sur 400 en 250 s → 0,4 pas/s ; apprentissage 1 000 s, mesure 70 s.
      const d = { ...base, epoque: 4, etape: 'apprentissage', pas: 100, pas_par_epoque: 400, debut_epoque: '2026-10-27T12:55:50Z', duree_epoque_s: 1070, etat, journal }
      const e = estimer(d, T)
      expect(e.epoqueCourante).toBe(4)
      expect(e.resteEpoque).toBeCloseTo(300 / 0.4 + 70)
      expect(e.resteMax).toBeCloseTo(820 + 11 * 1070 + 140)
      expect(e.epoqueArret).toBe(7)   // meilleure à l'époque 2, patience 5
      expect(e.resteMin).toBeCloseTo(820 + 3 * 1070 + 140)
    })

    it('vitesse observée par la page prioritaire ; temps écoulé depuis la dernière nouvelle déduit', () => {
      const d = { ...base, epoque: 4, etape: 'apprentissage', pas: 100, pas_par_epoque: 400, debut_epoque: '2026-10-27T12:55:50Z', duree_epoque_s: 1070, etat, journal }
      expect(estimer(d, T, { pas: 0.5, images: null }).resteEpoque).toBeCloseTo(300 / 0.5 + 800 * 0.07)
      expect(estimer(d, T + 60_000).resteEpoque).toBeCloseTo(820 - 60)
    })

    it('sans etat.json : durée des époques passées, à partir de la dernière ligne du journal', () => {
      const d = { ...base, avec_etat: false, epoque: 3, duree_epoque_s: 1070, journal }
      const e = estimer(d, T + 600_000)
      expect(e.epoqueCourante).toBe(4)
      expect(e.resteEpoque).toBeCloseTo(470)
    })

    it('prétraitement, phases sans estimation, entraînements finis', () => {
      const pre = { ...base, phase: 'pretraitement', debut_phase: '2026-10-27T12:58:20Z', etat: { pretraites: 500, a_pretraiter: 3000 } }
      expect(estimer(pre, T).restePretraitement).toBeCloseTo(500)   // 5 images/s
      expect(estimer({ ...base, phase: 'calibrage' }, T).resteMax).toBeNull()
      expect(estimer({ ...base, statut: 'termine' }, T).resteMax).toBeNull()
    })

    it('vitesse observée et heure de fin', () => {
      const r = (t: number, pas: number, epoque = 4) => ({ t: T + t * 1000, phase: 'entrainement', epoque, pas, pretraites: 0 })
      expect(vitesseObservee([r(0, 10), r(5, 12)]).pas).toBeNull()          // moins de 15 s
      expect(vitesseObservee([r(0, 10), r(20, 20)]).pas).toBeCloseTo(0.5)
      expect(vitesseObservee([r(0, 300, 3), r(20, 2), r(40, 12)]).pas).toBeCloseTo(0.5)   // autre époque écartée
      expect(heureFin(T, 0)).toMatch(/\d{2}:\d{2}/)
      expect(heureFin(T, 3 * 86400)).toMatch(/^\S+ \d{2}:\d{2}$/)
    })
  })

  it('relancer : reprise exacte, nouvel entraînement, ou impossible', () => {
    expect(relance({ statut: 'en_cours', avec_parametres: true }).possible).toBe(false)
    expect(relance({ statut: 'interrompu', avec_parametres: true }).libelle).toBe('Reprendre')
    expect(relance({ statut: 'sans_nouvelles', avec_parametres: true }).libelle).toBe('Reprendre')
    expect(relance({ statut: 'termine', avec_parametres: true })).toMatchObject({ possible: true, libelle: 'Relancer' })
    expect(relance({ statut: 'sans_nouvelles', avec_parametres: false }).explication).toMatch(/partira de zéro/)
  })

  it('chaque action de l\'API et du veilleur a un libellé', () => {
    const go = readFileSync(resolve(racine, '../backend/internal/entrainements/demandes.go'), 'utf8')
    const actions = [...go.matchAll(/(?:Relancer|Supprimer|Pause)\s+= "([a-z]+)"/g)].map(m => m[1]!)
    expect(actions).toEqual(['relancer', 'supprimer', 'pause'])
    const auto = readFileSync(resolve(racine, '../engine/mammomesh_engine/cnn/auto.py'), 'utf8')
    for (const a of actions) {
      expect(libelleDemande[a], a).toBeDefined()
      expect(auto, a).toContain(`action == "${a}"`)
    }
  })

  it('nouvel entraînement : mêmes réglages dans l\'interface, l\'API et le veilleur', () => {
    const go = readFileSync(resolve(racine, '../backend/internal/entrainements/demandes.go'), 'utf8')
    const champsGo = [...go.slice(go.indexOf('type Reglages struct')).split('}')[0]!.matchAll(/json:"([a-z_]+)"/g)].map(m => m[1]!)
    expect(Object.keys(aideReglages).sort()).toEqual(champsGo.sort())
    const auto = readFileSync(resolve(racine, '../engine/mammomesh_engine/cnn/auto.py'), 'utf8')
    const bloc = auto.slice(auto.indexOf('REGLAGES_INTERFACE = {')).split('}')[0]!
    for (const k of champsGo) expect(bloc, k).toContain(`"${k}"`)
    expect(go).toMatch(/Nouveau\s+= "nouveau"/)
    expect(auto).toContain('action == "nouveau"')
  })

  it('pause : seulement en cours ou en attente, avec un point de reprise', () => {
    expect(pausePossible({ statut: 'en_cours', avec_parametres: true }).possible).toBe(true)
    expect(pausePossible({ statut: 'en_attente_gpu', avec_parametres: true }).possible).toBe(true)
    expect(pausePossible({ statut: 'en_cours', avec_parametres: false }).possible).toBe(false)
    expect(pausePossible({ statut: 'en_cours', avec_parametres: true, pause_demandee: true }).possible).toBe(false)
    expect(pausePossible({ statut: 'en_pause', avec_parametres: true }).possible).toBe(false)
    expect(relance({ statut: 'en_pause', avec_parametres: true }).libelle).toBe('Reprendre')
    expect(estFinalise('en_pause')).toBe(false)
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
    for (const [, s] of go.matchAll(/(?:EnCours|Termine|Echec|Interrompu|SansNouvelle|EnAttente|EnPause)\s+= "([a-z_]+)"/g)) expect(statutEntrainement[s!], s).toBeDefined()
    const auto = readFileSync(resolve(racine, '../engine/mammomesh_engine/cnn/auto.py'), 'utf8')
    for (const [, s] of auto.matchAll(/(?:EN_ATTENTE = |EN_PAUSE = |statut=)"([a-z_]+)"/g)) expect(statutEntrainement[s!], s).toBeDefined()
    expect(estFinalise('en_attente_gpu')).toBe(false)
    const py = ['suivi.py', 'entrainement.py', '__main__.py'].map(f => readFileSync(resolve(racine, `../engine/mammomesh_engine/cnn/${f}`), 'utf8')).join('\n')
    const phases = [...py.matchAll(/phase\("([a-z_]+)"/g), ...py.matchAll(/"phase": "([a-z_]+)"/g)].map(m => m[1]!)
    expect(phases.length).toBeGreaterThan(4)
    for (const ph of phases) expect(phaseEntrainement[ph], ph).toBeDefined()
    for (const st of [...py.matchAll(/fin\("([a-z_]+)"/g)].map(m => m[1]!)) expect(statutEntrainement[st], st).toBeDefined()
  })
})
