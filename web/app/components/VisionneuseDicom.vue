<script setup lang="ts">
// Visionneuse de mammographie : pixels 16 bits reçus du serveur, fenêtrage (centre / largeur),
// inversion, zoom à la molette, déplacement à la souris, zones d'un test superposées.
// Souris : glisser = déplacer ; clic droit glissé (ou Maj + glisser) = fenêtrage ; double-clic = ajuster.
// Clavier (visionneuse sélectionnée) : + et − zoom, 0 ajuster, i inverser, f fenêtre d'origine, z zones.
import type { ImageMeta, Zone } from '~/types'

const props = defineProps<{ meta: ImageMeta; pixels: Uint16Array | null; zones?: Zone[]; zoneActive?: number | null }>()
const emit = defineEmits<{ zone: [i: number | null] }>()

const ecran = ref<HTMLCanvasElement | null>(null)
const cadre = ref<HTMLDivElement | null>(null)
const centre = ref(props.meta.window_center)
const largeur = ref(props.meta.window_width)
const inverse = ref(props.meta.invert)
const zonesVisibles = ref(true)
const vue = reactive({ echelle: 1, x: 0, y: 0 })
let base: HTMLCanvasElement | null = null // image fenêtrée, en pixels réels
let rafPrevu = false

const plageMax = computed(() => Math.max(props.meta.max, 1))

function fenetrer() {
  if (!props.pixels) return
  const { cols, rows } = props.meta
  if (!base) {
    base = document.createElement('canvas')
  }
  base.width = cols
  base.height = rows
  const ctx = base.getContext('2d')!
  const img = ctx.createImageData(cols, rows)
  // Table de correspondance : une valeur 16 bits → un niveau de gris 8 bits.
  const lut = new Uint8ClampedArray(plageMax.value + 1)
  const bas = centre.value - largeur.value / 2
  for (let v = 0; v < lut.length; v++) {
    const g = ((v - bas) / largeur.value) * 255
    lut[v] = inverse.value ? 255 - g : g
  }
  const px = props.pixels
  const out = new Uint32Array(img.data.buffer) // RGBA en une écriture (petit-boutiste : ABGR)
  for (let i = 0; i < px.length; i++) {
    const g = lut[px[i]!]!
    out[i] = 0xff000000 | (g << 16) | (g << 8) | g
  }
  ctx.putImageData(img, 0, 0)
  dessiner()
}

function dessiner() {
  const c = ecran.value
  if (!c || !base) return
  const ctx = c.getContext('2d')!
  const dpr = window.devicePixelRatio || 1
  ctx.setTransform(1, 0, 0, 1, 0, 0)
  ctx.fillStyle = '#000'
  ctx.fillRect(0, 0, c.width, c.height)
  ctx.setTransform(vue.echelle * dpr, 0, 0, vue.echelle * dpr, vue.x * dpr, vue.y * dpr)
  ctx.imageSmoothingEnabled = vue.echelle < 1
  ctx.drawImage(base, 0, 0)
  if (zonesVisibles.value && props.zones?.length) {
    props.zones.forEach((z, i) => {
      const actif = props.zoneActive === i
      ctx.lineWidth = (actif ? 4 : 2) / vue.echelle
      ctx.strokeStyle = actif ? '#ffd166' : '#ff5c5c'
      ctx.strokeRect(z.x, z.y, z.largeur, z.hauteur)
      ctx.font = `${14 / vue.echelle}px system-ui, sans-serif`
      ctx.fillStyle = ctx.strokeStyle
      ctx.fillText(`${i + 1} · ${z.score.toFixed(2)}`, z.x, z.y - 4 / vue.echelle)
    })
  }
}

function planifier() {
  if (rafPrevu) return
  rafPrevu = true
  requestAnimationFrame(() => { rafPrevu = false; fenetrer() })
}

function ajuster() {
  const c = cadre.value
  if (!c) return
  const l = c.clientWidth
  const h = c.clientHeight
  vue.echelle = Math.min(l / props.meta.cols, h / props.meta.rows)
  vue.x = (l - props.meta.cols * vue.echelle) / 2
  vue.y = (h - props.meta.rows * vue.echelle) / 2
  dessiner()
}

function zoomer(facteur: number, cx?: number, cy?: number) {
  const c = cadre.value!
  const px = cx ?? c.clientWidth / 2
  const py = cy ?? c.clientHeight / 2
  const e = Math.min(16, Math.max(0.05, vue.echelle * facteur))
  vue.x = px - ((px - vue.x) * e) / vue.echelle
  vue.y = py - ((py - vue.y) * e) / vue.echelle
  vue.echelle = e
  dessiner()
}

function unPourUn() { zoomer(1 / vue.echelle) }

function fenetreOrigine() {
  centre.value = props.meta.window_center
  largeur.value = props.meta.window_width
  inverse.value = props.meta.invert
}

// ---------- Souris et clavier ----------
let glisse: { x: number; y: number; mode: 'deplacer' | 'fenetre'; c: number; l: number; vx: number; vy: number } | null = null
function down(e: PointerEvent) {
  ecran.value?.focus()
  ;(e.target as HTMLElement).setPointerCapture(e.pointerId)
  glisse = { x: e.clientX, y: e.clientY, mode: e.button === 2 || e.shiftKey ? 'fenetre' : 'deplacer', c: centre.value, l: largeur.value, vx: vue.x, vy: vue.y }
}
function move(e: PointerEvent) {
  if (!glisse) return
  const dx = e.clientX - glisse.x
  const dy = e.clientY - glisse.y
  if (glisse.mode === 'deplacer') {
    vue.x = glisse.vx + dx
    vue.y = glisse.vy + dy
    dessiner()
  } else {
    const pas = plageMax.value / 600
    largeur.value = Math.max(1, glisse.l + dx * pas)
    centre.value = glisse.c + dy * pas
  }
}
function up() { glisse = null }
function wheel(e: WheelEvent) {
  const r = ecran.value!.getBoundingClientRect()
  zoomer(e.deltaY < 0 ? 1.15 : 1 / 1.15, e.clientX - r.left, e.clientY - r.top)
}
function touche(e: KeyboardEvent) {
  const actions: Record<string, () => void> = {
    '+': () => zoomer(1.25), '=': () => zoomer(1.25), '-': () => zoomer(0.8), '0': ajuster,
    i: () => { inverse.value = !inverse.value }, f: fenetreOrigine, z: () => { zonesVisibles.value = !zonesVisibles.value },
  }
  const a = actions[e.key]
  if (a) { e.preventDefault(); a() }
}
function survol(e: PointerEvent) {
  if (glisse || !props.zones?.length) return
  const r = ecran.value!.getBoundingClientRect()
  const ix = (e.clientX - r.left - vue.x) / vue.echelle
  const iy = (e.clientY - r.top - vue.y) / vue.echelle
  const i = props.zones.findIndex(z => ix >= z.x && ix <= z.x + z.largeur && iy >= z.y && iy <= z.y + z.hauteur)
  emit('zone', i >= 0 ? i : null)
}

// ---------- Taille du canevas ----------
let obs: ResizeObserver | null = null
function redimensionner() {
  const c = ecran.value
  const f = cadre.value
  if (!c || !f) return
  const dpr = window.devicePixelRatio || 1
  c.width = Math.round(f.clientWidth * dpr)
  c.height = Math.round(f.clientHeight * dpr)
  dessiner()
}
onMounted(() => {
  obs = new ResizeObserver(() => { redimensionner() })
  if (cadre.value) obs.observe(cadre.value)
  redimensionner()
  if (props.pixels) { fenetrer(); ajuster() }
})
onBeforeUnmount(() => obs?.disconnect())

watch(() => props.pixels, (p) => { if (p) { fenetreOrigine(); fenetrer(); ajuster() } })
watch([centre, largeur, inverse], planifier)
watch([() => props.zones, () => props.zoneActive, zonesVisibles], dessiner, { deep: true })

const zoomPct = computed(() => Math.round(vue.echelle * 100))
</script>

<template>
  <div class="visionneuse">
    <div class="outils row" role="toolbar" aria-label="Outils de la visionneuse">
      <button type="button" class="btn btn-sm" title="Ajuster à la fenêtre (0)" @click="ajuster">Ajuster</button>
      <button type="button" class="btn btn-sm" title="Taille réelle" @click="unPourUn">1:1</button>
      <button type="button" class="btn btn-sm" title="Zoom avant (+)" aria-label="Zoom avant" @click="zoomer(1.25)">+</button>
      <button type="button" class="btn btn-sm" title="Zoom arrière (−)" aria-label="Zoom arrière" @click="zoomer(0.8)">−</button>
      <span class="small muted zoom">{{ zoomPct }} %</span>
      <button type="button" class="btn btn-sm" :aria-pressed="inverse" title="Inverser les niveaux de gris (i)" @click="inverse = !inverse">Inverser</button>
      <button type="button" class="btn btn-sm" title="Fenêtre d'origine (f)" @click="fenetreOrigine">Fenêtre d'origine</button>
      <button v-if="zones?.length" type="button" class="btn btn-sm" :aria-pressed="zonesVisibles" title="Afficher les zones (z)" @click="zonesVisibles = !zonesVisibles">Zones</button>
    </div>
    <div ref="cadre" class="cadre">
      <canvas ref="ecran" tabindex="0" aria-label="Image de mammographie : glisser pour déplacer, clic droit glissé pour le fenêtrage, molette pour zoomer"
        @pointerdown="down" @pointermove="(e) => { move(e); survol(e) }" @pointerup="up" @pointercancel="up"
        @wheel.prevent="wheel" @dblclick="ajuster" @contextmenu.prevent @keydown="touche" />
      <p v-if="!pixels" class="chargement">Chargement des pixels…</p>
    </div>
    <div class="fenetrage">
      <label>Centre <input v-model.number="centre" type="range" :min="0" :max="plageMax" step="1" aria-label="Centre de la fenêtre"> <span class="small">{{ Math.round(centre) }}</span></label>
      <label>Largeur <input v-model.number="largeur" type="range" :min="1" :max="plageMax" step="1" aria-label="Largeur de la fenêtre"> <span class="small">{{ Math.round(largeur) }}</span></label>
    </div>
  </div>
</template>

<style scoped>
.visionneuse { display: grid; gap: 8px; }
.outils { gap: 6px; }
.btn-sm { padding: 4px 10px; font-size: 0.85rem; }
.btn[aria-pressed='true'] { background: var(--accent-soft); border-color: var(--accent); }
.zoom { min-width: 48px; text-align: center; font-variant-numeric: tabular-nums; }
.cadre { position: relative; height: min(72vh, 820px); min-height: 320px; background: #000; border-radius: 8px; overflow: hidden; touch-action: none; }
canvas { width: 100%; height: 100%; display: block; cursor: grab; outline: none; }
canvas:focus-visible { box-shadow: inset 0 0 0 2px var(--accent); }
canvas:active { cursor: grabbing; }
.chargement { position: absolute; inset: 0; display: grid; place-items: center; color: #bbb; margin: 0; }
.fenetrage { display: flex; flex-wrap: wrap; gap: 6px 18px; }
.fenetrage label { display: flex; align-items: center; gap: 8px; margin: 0; }
.fenetrage input { width: 180px; padding: 0; }
</style>
