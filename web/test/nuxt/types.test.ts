import { describe, expect, it } from 'vitest'
import { fr, roleLabel, stateLabel } from '~/types'

describe('fr', () => {
  it('formate à la belge, virgule décimale', () => {
    expect(fr(0.8125)).toBe('0,813')
    expect(fr(12.5, 1)).toBe('12,5')
  })
  it('affiche un tiret pour une valeur absente', () => {
    expect(fr(null)).toBe('—')
    expect(fr(undefined)).toBe('—')
  })
})

describe('libellés', () => {
  it('couvre chaque état de campagne et chaque rôle', () => {
    expect(Object.keys(stateLabel).sort()).toEqual(['arretee', 'en_cours', 'en_echec', 'en_pause', 'terminee'])
    expect(Object.keys(roleLabel)).toHaveLength(5)
  })
})
