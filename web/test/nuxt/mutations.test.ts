import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { describe, expect, it } from 'vitest'
import { decouperMessage, expliquer, infobulle, mutations } from '~/utils/mutations'
import { parties } from '~/utils/fonctionnement'

const racine = resolve(__dirname, '../..')

describe('lexique des mutations', () => {
  it('chaque code produit par l\'orchestrateur a son explication', () => {
    const sim = readFileSync(resolve(racine, '../backend/internal/orchestrator/sim.go'), 'utf8')
    const bloc = sim.match(/var mutationCodes = \[\]string\{([^}]+)\}/)![1]!
    const codes = [...bloc.matchAll(/"([^"]+)"/g)].map(m => m[1]!)
    expect(codes.length).toBeGreaterThan(10)
    for (const c of codes) expect(expliquer(c), c).toBeDefined()
  })

  it('reconnaît les variantes de la note de conception', () => {
    for (const c of ['SYM−', 'TMP↓D', 'CUB12×8', 'TET-SA', 'X(ORION-G02.11 × LYRA-G04.03)']) expect(expliquer(c), c).toBeDefined()
    expect(expliquer('ZZZ')).toBeUndefined()
    expect(infobulle('ZZZ')).toBe('Mutation sans description')
    expect(infobulle('BUD↑')).toContain('Budget d\'activité')
  })

  it('découpe un message du journal autour des codes', () => {
    const msg = 'Génération 3 : nouveau meilleur compromis, ORION-G03.02, sensibilité 0,850 ; mutations SEU↑ › BLK+.'
    const parts = decouperMessage(msg)
    expect(parts.filter(p => p.code).map(p => p.code)).toEqual(['SEU↑', 'BLK+'])
    expect(parts.map(p => p.code ?? p.texte).join('')).toBe(msg)
    expect(decouperMessage('Génération 4 terminée.')).toEqual([{ texte: 'Génération 4 terminée.' }])
  })
})

describe('fonctionnement d\'une campagne', () => {
  const doc = readFileSync(resolve(racine, '../docs/fonctionnement-campagne.md'), 'utf8')
  it('le document du dépôt reprend mot pour mot la page', () => {
    for (const p of parties) {
      expect(doc).toContain(`## ${p.titre}`)
      for (const b of p.blocs) {
        for (const t of [b.p, ...(b.etapes ?? []), ...(b.points ?? []), ...(b.table?.lignes.map(l => `| ${l.join(' | ')} |`) ?? [])]) {
          if (t) expect(doc, t.slice(0, 40)).toContain(t)
        }
      }
    }
  })
  it('et le lexique complet des mutations', () => {
    for (const m of mutations) expect(doc).toContain(`| \`${m.famille}\` | ${m.partie} | ${m.gene} | ${m.role} | ${m.effet} |`)
  })
})

describe('modèle mathématique et guide Images', async () => {
  const { partiesMath } = await import('~/utils/mathematique')
  const { zonesImages } = await import('~/utils/guideImages')
  const katex = (await import('katex')).default
  it('chaque formule se compile sans erreur KaTeX', () => {
    for (const p of partiesMath) for (const b of p.blocs) {
      if (b.formule) expect(() => katex.renderToString(b.formule!, { displayMode: true, throwOnError: true, strict: 'ignore' }), b.formule).not.toThrow()
    }
  })
  it('les documents du dépôt reprennent mot pour mot les pages', () => {
    const math = readFileSync(resolve(racine, '../docs/modele-mathematique.md'), 'utf8')
    for (const p of partiesMath) {
      expect(math).toContain(`## ${p.titre}`)
      for (const b of p.blocs) {
        if (b.p) expect(math).toContain(b.p)
        if (b.formule) expect(math).toContain(`$$\n${b.formule}\n$$`)
      }
    }
    const img = readFileSync(resolve(racine, '../docs/guide-images.md'), 'utf8')
    for (const z of zonesImages) for (const e of z.elements) expect(img).toContain(`| ${e.nom} | ${e.description} |`)
  })
})
