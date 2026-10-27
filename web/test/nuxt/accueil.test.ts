import { existsSync, readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { describe, expect, it } from 'vitest'
import { comparaison, menus } from '~/utils/accueil'

const racine = resolve(__dirname, '../..')
const page = (to: string) => {
  const r = resolve(racine, 'app/pages', `.${to}`)
  return existsSync(`${r}.vue`) || existsSync(resolve(r, 'index.vue'))
}

describe('page d\'accueil', () => {
  it('chaque lien du menu (et Mon compte) est expliqué sur l\'accueil', () => {
    const layout = readFileSync(resolve(racine, 'app/layouts/suivi.vue'), 'utf8')
    const liens = [...layout.matchAll(/<NuxtLink[^>]*to="(\/suivi[^"]*)"/g)].map(m => m[1]!).filter(l => l !== '/suivi')
    expect(liens.length).toBeGreaterThan(5)
    for (const l of liens) expect(menus.map(m => m.to), l).toContain(l)
  })

  it('les liens de l\'accueil mènent à des pages qui existent', () => {
    for (const m of menus) {
      expect(page(m.to), m.to).toBe(true)
      for (const l of m.liens) expect(page(l.to), l.to).toBe(true)
      expect(m.fonctionnement.length, m.cle).toBeGreaterThan(1)
    }
  })

  it('les menus réservés suivent les mêmes permissions que la barre de navigation', () => {
    const layout = readFileSync(resolve(racine, 'app/layouts/suivi.vue'), 'utf8')
    for (const m of menus.filter(x => x.permission)) {
      expect(layout, m.to).toMatch(new RegExp(`permissions\\?\\.${m.permission}" to="${m.to}"`))
    }
  })

  it('la comparaison campagne / entraînement est complète', () => {
    expect(comparaison.length).toBeGreaterThan(5)
    for (const l of comparaison) expect(l.campagne && l.entrainement, l.aspect).toBeTruthy()
  })
})
