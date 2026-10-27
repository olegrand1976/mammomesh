import { existsSync, readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { describe, expect, it } from 'vitest'
import { comparaison, menus } from '~/utils/accueil'
import { entrees } from '~/utils/navigation'

const racine = resolve(__dirname, '../..')
const page = (to: string) => {
  const r = resolve(racine, 'app/pages', `.${to}`)
  return existsSync(`${r}.vue`) || existsSync(resolve(r, 'index.vue'))
}

describe('barre latérale et page d\'accueil', () => {
  it('la barre latérale vient de utils/navigation.ts et mène à des pages qui existent', () => {
    const layout = readFileSync(resolve(racine, 'app/layouts/suivi.vue'), 'utf8')
    expect(layout).toContain("from '~/utils/navigation'")
    expect(entrees[0]!.to).toBe('/suivi')
    for (const e of entrees) expect(page(e.to), e.to).toBe(true)
  })

  it('l\'accueil explique chaque entrée du menu, dans le même ordre, avec les mêmes permissions, puis Mon compte', () => {
    const sansAccueil = entrees.filter(e => e.to !== '/suivi')
    expect(menus.map(m => m.to)).toEqual([...sansAccueil.map(e => e.to), '/suivi/compte'])
    for (const e of sansAccueil) {
      const m = menus.find(x => x.to === e.to)!
      expect(m.titre, e.to).toBe(e.titre)
      expect(m.permission, e.to).toBe(e.permission)
    }
  })

  it('les liens de l\'accueil mènent à des pages qui existent', () => {
    for (const m of menus) {
      expect(page(m.to), m.to).toBe(true)
      for (const l of m.liens) expect(page(l.to), l.to).toBe(true)
      expect(m.fonctionnement.length, m.cle).toBeGreaterThan(1)
    }
  })

  it('chaque icône du menu est dessinée', () => {
    const icones = readFileSync(resolve(racine, 'app/components/IconeNav.vue'), 'utf8')
    for (const n of [...entrees.map(e => e.icone), 'compte', 'sortie', 'replier', 'deplier', 'menu', 'fermer']) expect(icones, n).toMatch(new RegExp(`\\b${n}: \\[`))
  })

  it('la comparaison campagne / entraînement est complète', () => {
    expect(comparaison.length).toBeGreaterThan(5)
    for (const l of comparaison) expect(l.campagne && l.entrainement, l.aspect).toBeTruthy()
  })
})
