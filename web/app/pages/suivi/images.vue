<script setup lang="ts">
// Images de test : import pseudonymisé de mammographies DICOM, visionneuse, test d'un individu.
import { fr, type ImageRow, type ImageTest } from '~/types'

definePageMeta({ layout: 'suivi', middleware: 'auth' })
useHead({ title: 'Images · MammoMesh' })

const { api } = useApi()
const session = useSessionStore()
const images = ref<ImageRow[]>([])
const choisie = ref<ImageRow | null>(null)
const pixels = ref<Uint16Array | null>(null)
const test = ref<ImageTest | null>(null)
const zoneActive = ref<number | null>(null)
const error = ref('')
const info = ref('')
const envoi = ref(false)
const libelle = ref('')
const survol = ref(false)

async function charger(selection?: number) {
  images.value = (await api<{ images: ImageRow[] }>('/images')).images
  const id = selection ?? choisie.value?.id ?? images.value[0]?.id
  const im = images.value.find(i => i.id === id) ?? null
  if (im && im.id !== choisie.value?.id) await ouvrir(im)
  else if (!im) choisie.value = null
}
onMounted(() => { charger().catch(e => { error.value = (e as Error).message }) })

async function ouvrir(im: ImageRow) {
  choisie.value = im
  test.value = null
  pixels.value = null
  const r = await fetch(`/api/images/${im.id}/pixels`, { credentials: 'same-origin' })
  if (!r.ok) { error.value = 'Pixels indisponibles'; return }
  if (choisie.value?.id === im.id) pixels.value = new Uint16Array(await r.arrayBuffer())
}

async function importer(fichier: File | undefined) {
  if (!fichier) return
  error.value = ''
  info.value = ''
  envoi.value = true
  try {
    const fd = new FormData()
    fd.append('fichier', fichier)
    if (libelle.value.trim()) fd.append('libelle', libelle.value.trim())
    const r = await fetch('/api/images', { method: 'POST', body: fd, credentials: 'same-origin' })
    const body = await r.json().catch(() => ({}))
    if (!r.ok) throw new Error(body.erreur ?? `Import refusé (${r.status})`)
    info.value = body.existe ? 'Cette image était déjà importée : elle est sélectionnée.' : 'Image importée, en-têtes identifiants écartés.'
    libelle.value = ''
    await charger(body.image.id)
  } catch (e) {
    error.value = (e as Error).message
  } finally {
    envoi.value = false
  }
}

function deposer(e: DragEvent) {
  survol.value = false
  void importer(e.dataTransfer?.files?.[0])
}

async function supprimer(im: ImageRow) {
  if (!confirm(`Supprimer « ${im.libelle} » et ses ${im.nb_tests} test(s) ? Cette action est définitive.`)) return
  await api(`/images/${im.id}`, { method: 'DELETE' })
  if (choisie.value?.id === im.id) { choisie.value = null; pixels.value = null }
  await charger()
}

const taille = (o: number) => o > 1e6 ? `${fr(o / 1e6, 1)} Mo` : `${fr(o / 1e3, 0)} ko`
</script>

<template>
  <div>
    <div class="row head">
      <h1>Images</h1>
      <span class="research-tag">recherche uniquement</span>
    </div>
    <p v-if="error" class="alert" role="alert">{{ error }}</p>
    <p v-if="info" class="notice" role="status">{{ info }}</p>
    <div class="disposition">
      <aside class="grid colonne">
        <section v-if="session.permissions?.tester_images" class="card depot" :class="{ survol }"
          @dragover.prevent="survol = true" @dragleave="survol = false" @drop.prevent="deposer">
          <h2>Importer une mammographie</h2>
          <div class="field">
            <label for="img-libelle">Libellé (facultatif)</label>
            <input id="img-libelle" v-model="libelle" placeholder="par exemple : essai MLO gauche" maxlength="200">
          </div>
          <label class="btn btn-primary fichier" :class="{ off: envoi }">
            {{ envoi ? 'Import…' : 'Choisir un fichier DICOM' }}
            <input type="file" accept=".dcm,application/dicom" :disabled="envoi" @change="(e) => importer((e.target as HTMLInputElement).files?.[0])">
          </label>
          <p class="small muted">Fichier <Sigle t="DICOM" />, ou le déposer ici. Rien d'identifiant n'est gardé : ni nom, ni identifiant, ni dates, ni établissement, ni nom de fichier ; seulement les pixels et des données techniques (dimensions, vue, latéralité, fabricant). Images à annotations incrustées refusées. Formats lus : non compressé et <Sigle t="JPEG" /> de base.</p>
          <p class="small">Pas d'image sous la main ? Exemples synthétiques (vues <Sigle t="CC" /> et <Sigle t="MLO" />) :
            <a href="/api/images/exemple.dcm?vue=CC&cote=L" download>CC gauche</a> ·
            <a href="/api/images/exemple.dcm?vue=MLO&cote=L" download>MLO gauche</a> ·
            <a href="/api/images/exemple.dcm?vue=CC&cote=R" download>CC droite</a> ·
            <a href="/api/images/exemple.dcm?vue=MLO&cote=R" download>MLO droite</a></p>
        </section>
        <section class="card">
          <h2>Images importées</h2>
          <p v-if="!images.length" class="muted small">Aucune image pour l'instant.</p>
          <ul class="liste">
            <li v-for="im in images" :key="im.id" :class="{ actif: choisie?.id === im.id }">
              <button type="button" class="choix" :aria-current="choisie?.id === im.id" @click="ouvrir(im)">
                <strong>{{ im.libelle }}</strong>
                <span class="small muted">{{ im.meta.cols }} × {{ im.meta.rows }} · <TexteGlossaire :texte="im.meta.view || '—'" /> {{ im.meta.laterality }} · {{ im.nb_tests }} test(s) · {{ taille(im.taille_octets) }}</span>
              </button>
              <button v-if="session.permissions?.purger_images" type="button" class="btn btn-sm btn-danger" :aria-label="`Supprimer ${im.libelle}`" @click="supprimer(im)">Supprimer</button>
            </li>
          </ul>
        </section>
      </aside>
      <section v-if="choisie" class="card visu">
        <div class="row">
          <h2>{{ choisie.libelle }}</h2>
          <div class="spacer" />
          <span class="small muted">{{ choisie.meta.modality }} · {{ choisie.meta.manufacturer || 'fabricant inconnu' }} · {{ choisie.meta.bits_stored }} bits · {{ choisie.meta.transfer_syntax }}</span>
        </div>
        <VisionneuseDicom :key="choisie.id" :meta="choisie.meta" :pixels="pixels" :zones="test?.resultat.zones" :zone-active="zoneActive" @zone="zoneActive = $event" />
        <p class="small muted">Glisser : déplacer · clic droit glissé (ou Maj) : fenêtrage · molette : zoom · double-clic : ajuster · touches + − 0 i f z.</p>
      </section>
      <TestIndividu v-if="choisie" :image-id="choisie.id" class="colonne-test" @resultat="test = $event; zoneActive = null; charger()" />
    </div>
  </div>
</template>

<style scoped>
.head { margin-bottom: 14px; }
.head h1 { margin: 0; font-size: 1.4rem; }
.disposition { display: grid; gap: 16px; grid-template-columns: 280px minmax(0, 1fr) 300px; align-items: start; }
@media (max-width: 1100px) { .disposition { grid-template-columns: 1fr; } }
.colonne { gap: 16px; }
.depot.survol { outline: 2px dashed var(--accent); outline-offset: -6px; }
.fichier { position: relative; overflow: hidden; justify-content: center; width: 100%; }
.fichier input { position: absolute; inset: 0; opacity: 0; cursor: pointer; }
.fichier.off { opacity: 0.6; }
.liste { list-style: none; margin: 0; padding: 0; display: grid; gap: 6px; }
.liste li { display: flex; gap: 6px; align-items: center; border: 1px solid var(--border); border-radius: 8px; padding: 6px; }
.liste li.actif { border-color: var(--accent); background: var(--accent-soft); }
.choix { flex: 1; display: grid; text-align: left; gap: 2px; border: 0; background: none; color: inherit; font: inherit; cursor: pointer; padding: 2px; min-width: 0; }
.btn-sm { padding: 3px 8px; font-size: 0.8rem; }
.visu h2 { margin: 0; }
</style>
