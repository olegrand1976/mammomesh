import { describe, expect, it } from 'vitest'
import { mountSuspended } from '@nuxt/test-utils/runtime'
import { defineComponent, h, ref } from 'vue'
import ChampConfig from '~/components/ChampConfig.vue'

// Champ « Population » dans un parent qui tient la valeur (v-model).
const monter = async () => {
  const valeur = ref<string | number>(100)
  const w = await mountSuspended(defineComponent({
    setup: () => () => h(ChampConfig, {
      cle: 'evolution.population', defaut: 100, type: 'number', min: 20, max: 500,
      'modelValue': valeur.value, 'onUpdate:modelValue': (v: string | number) => { valeur.value = v },
    }),
  }))
  return { w, valeur }
}

describe('ChampConfig', () => {
  it('libellé, plage et défaut ; aide repliée par défaut', async () => {
    const { w } = await monter()
    expect(w.find('label').text()).toBe('Population')
    expect(w.find('input').attributes('min')).toBe('20')
    expect(w.find('.info').text()).toContain('20 à 500 individus')
    expect(w.find('.info').text()).toContain('défaut 100')
    expect(w.find('.aide').attributes('style')).toContain('display: none')
    expect(w.find('.aide-btn').attributes('aria-expanded')).toBe('false')
  })

  it('« ? » déplie le rôle et l\'effet d\'un changement', async () => {
    const { w } = await monter()
    await w.find('.aide-btn').trigger('click')
    expect(w.find('.aide-btn').attributes('aria-expanded')).toBe('true')
    expect(w.find('.aide').text()).toContain('À quoi ça sert.')
    expect(w.find('.aide').text()).toContain('cœur-heure par génération')
  })

  it('une saisie est un nombre, signalée « modifié » et rétablie d\'un clic', async () => {
    const { w, valeur } = await monter()
    await w.find('input').setValue('250')
    expect(valeur.value).toBe(250)
    await w.vm.$nextTick()
    expect(w.find('.info').text()).toContain('modifié')
    await w.find('.lien').trigger('click')
    expect(valeur.value).toBe(100)
  })
})
