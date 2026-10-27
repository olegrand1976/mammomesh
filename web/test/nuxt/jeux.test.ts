import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { describe, expect, it } from 'vitest'
import { etatJeux, partiesJeux, sourcesJeux } from '~/utils/jeuxEntrainement'

const racine = resolve(__dirname, '../..')

describe('jeux d\'entraînement', () => {
  const doc = readFileSync(resolve(racine, '../docs/jeux-entrainement.md'), 'utf8')

  it('le document du dépôt reprend mot pour mot la page', () => {
    expect(doc).toContain(etatJeux)
    for (const p of partiesJeux) {
      expect(doc).toContain(`## ${p.titre}`)
      for (const b of p.blocs) {
        for (const t of [b.p, ...(b.etapes ?? []), ...(b.points ?? []), ...(b.table?.lignes.map(l => `| ${l.join(' | ')} |`) ?? [])]) {
          if (t) expect(doc, t.slice(0, 40)).toContain(t)
        }
      }
    }
    for (const s of sourcesJeux) expect(doc).toContain(`- [${s.titre}](${s.url})`)
  })

  it('chaque collection téléchargeable du manifeste figure parmi les jeux retenus', () => {
    const manifeste = JSON.parse(readFileSync(resolve(racine, '../examples/manifeste.json'), 'utf8')) as { banque: { nom: string; licence: string }[] }
    const retenus = partiesJeux.find(p => p.id === 'retenus')!.blocs[0]!.table!.lignes
    expect(manifeste.banque.length).toBeGreaterThan(0)
    for (const c of manifeste.banque) {
      const ligne = retenus.find(l => l[0] === c.nom)
      expect(ligne, c.nom).toBeDefined()
      expect(ligne![3], c.nom).toBe(c.licence)
    }
  })

  it('les sources sont des adresses https', () => {
    for (const s of sourcesJeux) expect(s.url, s.titre).toMatch(/^https:\/\//)
  })
})
