import { readdirSync, readFileSync, statSync } from 'node:fs'
import { join, resolve } from 'node:path'
import { describe, expect, it } from 'vitest'
import { decouper, glossaire, terme } from '~/utils/glossaire'
import { partiesWindows } from '~/utils/installationWindows'
import { expliquer, mutations } from '~/utils/mutations'

const racine = resolve(__dirname, '../..')
const fichiers = (dossier: string, ext: string): string[] => readdirSync(dossier).flatMap((f) => {
  const p = join(dossier, f)
  return statSync(p).isDirectory() ? fichiers(p, ext) : p.endsWith(ext) ? [p] : []
})
const gabarit = (src: string) => src.slice(src.indexOf('<template>'), src.lastIndexOf('</template>'))

// Majuscules qui ne sont pas des sigles à expliquer : marques, noms propres, fragments techniques.
const HORS_GLOSSAIRE = new Set(['AI', 'AMD', 'NVIDIA', 'RTX', 'ORION', 'MOTIF', 'PATIENTE', 'COLLECTION', 'VERSION', 'YES', 'K9', 'NFD',
  'B0', 'II', 'BE', 'BY', 'MONOCHROME1', 'CCD', 'DDSM', 'CSAW', 'NL-Breast', 'HITI', 'OPTIMAM', 'HV', 'VOTRE_NOM', 'GET', 'POST',
  'ID', 'JSON', 'URL', 'CSS', 'SVG', 'HTML', 'UTC', 'RGBA', 'SC', 'CT', 'IRM', 'UE', 'FN', 'FP', 'TP', 'TN'])

describe('glossaire', () => {
  it('termes uniques, sigles développés, définitions complètes', () => {
    expect(new Set(glossaire.map(t => t.terme)).size).toBe(glossaire.length)
    for (const t of glossaire) {
      expect(t.definition, t.terme).toMatch(/\.$/)
      if (t.sigle) expect(t.forme, t.terme).toBeTruthy()
    }
  })

  it('repère les sigles entiers, le plus long d\'abord', () => {
    const m = decouper('CC BY 4.0 ; vue CC ; JPEG-LS refusé ; E-val et E-ajust ; CNN+CephaloMesh ; AUC.')
    expect(m.filter(x => x.sigle).map(x => x.texte)).toEqual(['CC BY', 'CC', 'JPEG-LS', 'E-val', 'E-ajust', 'CNN', 'AUC'])
    expect(decouper('ACCORD, CCD, AUCUN').some(x => x.sigle)).toBe(false)
    expect(decouper('a CNN b').map(x => x.texte).join('')).toBe('a CNN b')
  })

  it('chaque <Sigle t="…"> des écrans existe dans le glossaire', () => {
    for (const f of fichiers(resolve(racine, 'app'), '.vue')) {
      for (const [, t] of readFileSync(f, 'utf8').matchAll(/<Sigle t="([^"]+)"/g)) expect(terme(t!), `${f} : ${t}`).toBeDefined()
    }
  })

  it('chaque sigle des écrans et de leurs textes est expliqué', () => {
    const textes = [
      ...fichiers(resolve(racine, 'app/utils'), '.ts').filter(f => !f.endsWith('glossaire.ts')).map(f => readFileSync(f, 'utf8')),
      ...fichiers(resolve(racine, 'app'), '.vue').map(f => gabarit(readFileSync(f, 'utf8'))),
    ].join('\n').replace(/\{\{[\s\S]*?\}\}/g, ' ').replace(/(formule|motif|url|to|href|code): *'(?:[^'\\]|\\.)*'/g, ' ')
    const familles = new Set(mutations.flatMap(m => [m.famille, ...m.famille.split(/[^A-Z0-9]+/), m.famille.replace(/\d.*$/, '')]))
    const manquants = new Set<string>()
    for (const [m] of textes.matchAll(/(?<![\p{L}\p{N}_-])[A-Z][A-Z0-9]+(?:-[A-Za-z0-9]+)?(?![\p{L}\p{N}_-])/gu)) {
      // Codes de mutation (expliqués par leur propre lexique), noms d'individus (ORION-G03…), repères (M1…).
      if (!terme(m) && !HORS_GLOSSAIRE.has(m) && !expliquer(m) && !familles.has(m) && !familles.has(m.replace(/\d+$/, '')) && !/^ORION/.test(m) && !/^[A-Z]\d*$/.test(m) && !/^M\d+$/.test(m)) manquants.add(m)
    }
    expect([...manquants].sort()).toEqual([])
  })

  it('le document du dépôt reprend mot pour mot la page', () => {
    const doc = readFileSync(resolve(racine, '../docs/glossaire.md'), 'utf8')
    for (const t of glossaire) expect(doc, t.terme).toContain(`**${t.terme}**${t.forme ? ` (${t.forme})` : ''} : ${t.definition}`)
  })
})

describe('installation sur Windows 11', () => {
  const doc = readFileSync(resolve(racine, '../docs/installation-windows-11.md'), 'utf8')
  const md = (s: string) => s.replaceAll('\\', '\\\\')
  it('le document du dépôt reprend mot pour mot la page', () => {
    for (const p of partiesWindows) {
      expect(doc).toContain(`## ${p.titre}`)
      for (const b of p.blocs) {
        for (const t of [b.p, ...(b.etapes ?? []), ...(b.points ?? [])]) if (t) expect(doc, t.slice(0, 40)).toContain(md(t))
        for (const l of b.table?.lignes ?? []) expect(doc).toContain(`| ${l.map(md).join(' | ')} |`)
        if (b.code) expect(doc).toContain(b.code)
      }
    }
  })
  it('le PDF existe dans le dépôt et dans l\'interface, identique', () => {
    const a = readFileSync(resolve(racine, '../docs/installation-windows-11.pdf'))
    const b = readFileSync(resolve(racine, 'public/docs/installation-windows-11.pdf'))
    expect(a.subarray(0, 5).toString()).toBe('%PDF-')
    expect(a.equals(b)).toBe(true)
  })
})
