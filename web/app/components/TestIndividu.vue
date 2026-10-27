<script setup lang="ts">
// Choix d'un individu (campagne → génération, la finale par défaut → individu du front, le meilleur
// sur E-val par défaut) et test sur l'image affichée : score du CNN entraîné si le service d'inférence est
// branché (INFERENCE_URL), sinon résultat simulé.
import { fr, type Campaign, type EtatInference, type Generation, type ImageTest, type Individual } from '~/types'

const props = defineProps<{ imageId: number }>()
const emit = defineEmits<{ resultat: [t: ImageTest | null] }>()
const { api } = useApi()
const session = useSessionStore()

const campagnes = ref<Campaign[]>([])
const campagneId = ref<number | null>(null)
const generations = ref<Generation[]>([])
const generation = ref<number | null>(null)
const front = ref<Individual[]>([])
const individuId = ref<number | null>(null)
const historique = ref<ImageTest[]>([])
const courant = ref<ImageTest | null>(null)
const error = ref('')
const busy = ref(false)
const inference = ref<EtatInference | null>(null)
const reel = computed(() => inference.value?.mode === 'cnn')

const campagne = computed(() => campagnes.value.find(c => c.id === campagneId.value) ?? null)
const individu = computed(() => front.value.find(i => i.id === individuId.value) ?? null)

onMounted(async () => {
  api<EtatInference>('/inference').then(r => { inference.value = r }).catch(() => { inference.value = null })
  try {
    campagnes.value = (await api<{ campaigns: Campaign[] }>('/campaigns')).campaigns.filter(c => c.generation > 0)
    // Par défaut : la campagne finie la plus récente qui a des individus réalisables, sinon la plus récente.
    const finie = campagnes.value.find(c => (c.state === 'terminee' || c.state === 'arretee') && c.best_sensitivity !== null)
    campagneId.value = (finie ?? campagnes.value[0])?.id ?? null
  } catch (e) { error.value = (e as Error).message }
})

watch(campagneId, async (id) => {
  generations.value = []
  generation.value = null
  if (!id) return
  const liste = (await api<{ generations: Generation[] }>(`/campaigns/${id}/generations`)).generations
  if (campagneId.value !== id) return
  generations.value = liste
  // La finale par défaut ; si elle n'a aucun réalisable (donc pas de front), la dernière qui en a.
  const finale = campagne.value?.generation
  const avecFront = [...liste].reverse().find(g => g.feasible > 0)
  generation.value = liste.find(g => g.number === finale && g.feasible > 0)?.number ?? avecFront?.number ?? null
})

watch([campagneId, generation], async ([id, g]) => {
  front.value = []
  individuId.value = null
  if (!id || !g) return
  const liste = (await api<{ front: Individual[] }>(`/campaigns/${id}/front`, { query: { generation: g } })).front
  if (campagneId.value !== id || generation.value !== g) return // réponse dépassée par un autre choix
  front.value = liste
  const meilleur = [...front.value].sort((a, b) => b.sensitivity_eval - a.sensitivity_eval)[0]
  individuId.value = meilleur?.id ?? null // le meilleur sur E-val par défaut
})

async function chargerHistorique() {
  historique.value = (await api<{ tests: ImageTest[] }>(`/images/${props.imageId}/tests`)).tests
}
watch(() => props.imageId, () => { courant.value = null; emit('resultat', null); void chargerHistorique() }, { immediate: true })

async function tester() {
  if (!individuId.value) return
  error.value = ''
  busy.value = true
  try {
    const t = (await api<{ test: ImageTest }>(`/images/${props.imageId}/tests`, { method: 'POST', body: { individual_id: individuId.value } })).test
    courant.value = t
    emit('resultat', t)
    await chargerHistorique()
  } catch (e) { error.value = (e as Error).message } finally { busy.value = false }
}

function revoir(t: ImageTest) {
  courant.value = t
  emit('resultat', t)
}

const libelleGen = (g: Generation) =>
  `Génération ${g.number}${g.number === campagne.value?.generation ? ' (finale)' : ''} · meilleure ${fr(g.best_sensitivity)}`
const niveau = (s: number) => (s < 0.3 ? 'peu suspect' : s < 0.6 ? 'à surveiller' : 'suspect')
const typesZone: Record<string, string> = {
  masse: 'une grosseur',
  microcalcifications: 'minuscules dépôts de calcium',
  'distorsion architecturale': 'le tissu paraît tiré ou déformé',
  asymétrie: 'zone plus dense que du côté opposé',
  'attention du CNN': 'région qui a le plus pesé dans le score',
}
</script>

<template>
  <section class="card test" aria-labelledby="test-titre">
    <h2 id="test-titre">Tester un individu</h2>
    <p class="small">
      Un individu est une version de MammoMesh trouvée par une campagne. Le tester ici montre ce qu'il signale sur cette image, et permet de comparer
      plusieurs versions sur le même cas (par exemple la plus sûre et la plus rapide).
    </p>
    <p class="small muted">Par défaut : la dernière campagne terminée, sa génération finale et, parmi ses meilleurs compromis (le front), la version qui trouve le plus de cancers sur les images de contrôle (<Sigle t="E-val" />).</p>
    <p v-if="error" class="alert">{{ error }}</p>
    <p v-if="!campagnes.length" class="muted small">Aucune campagne avec des générations calculées.</p>
    <template v-else>
      <div class="field">
        <label for="t-campagne">Campagne</label>
        <select id="t-campagne" v-model.number="campagneId">
          <option v-for="c in campagnes" :key="c.id" :value="c.id">{{ c.name }} ({{ c.generation }} générations)</option>
        </select>
      </div>
      <div class="field">
        <label for="t-generation">Génération</label>
        <select id="t-generation" v-model.number="generation">
          <option v-for="g in [...generations].reverse()" :key="g.number" :value="g.number" :disabled="!g.feasible">{{ libelleGen(g) }}</option>
        </select>
      </div>
      <div class="field">
        <label for="t-individu">Individu (parmi les meilleurs compromis)</label>
        <p v-if="generation === null && generations.length" class="small muted">Aucune génération de cette campagne n'a d'individu réalisable.</p>
        <select id="t-individu" v-model.number="individuId" :disabled="!front.length">
          <option v-for="i in front" :key="i.id" :value="i.id">{{ i.name }} · {{ Math.round(i.sensitivity_eval * 100) }} cancers sur 100 · {{ fr(i.cost_ms, 0) }} ms</option>
        </select>
      </div>
      <p v-if="individu" class="small muted">Sensibilité en contrôle (<Sigle t="E-val" />) : {{ fr(individu.sensitivity_eval) }}, soit {{ Math.round(individu.sensitivity_eval * 100) }} cancers trouvés sur 100 ; {{ fr(individu.cost_ms, 1) }} ms de calcul par examen.</p>
      <p v-if="individu" class="small muted">Mutations qui l'ont produit : <template v-for="(m, k) in individu.mutations" :key="k"><template v-if="k"> › </template><MutationCode :code="m" /></template></p>
      <button v-if="session.permissions?.tester_images" type="button" class="btn btn-primary" :disabled="busy || !individuId" @click="tester">
        {{ busy ? (reel ? 'Calcul du CNN…' : 'Test…') : reel ? 'Tester sur cette image (CNN)' : 'Tester sur cette image (simulation)' }}
      </button>
      <p v-if="reel && inference?.joignable && inference.etat" class="small muted">
        Score donné par le <Sigle t="CNN" /> entraîné « {{ inference.etat.modele }} »<template v-if="inference.etat.epoque"> (époque {{ inference.etat.epoque }})</template>{{ inference.etat.fini ? '' : ', entraînement encore en cours' }}.
        CephaloMesh n'est pas encore branché : l'individu choisi n'influence pas encore le score.
      </p>
      <p v-else-if="reel" class="alert small">Moteur d'inférence injoignable : vérifiez le service mammomesh-inference.<template v-if="inference?.erreur"> ({{ inference.erreur }})</template></p>
      <p v-else class="notice small">Votre rôle permet de consulter les tests, pas d'en lancer.</p>
    </template>

    <div v-if="courant" class="resultat" aria-live="polite">
      <p class="simul" :class="{ reel: !courant.resultat.simulation }" role="note">{{ courant.resultat.avertissement }}</p>
      <template v-if="courant.resultat.simulation">
        <p><strong>{{ courant.individu }}</strong> (campagne {{ courant.campaign_id }}, génération {{ courant.generation }}) :
          score de suspicion {{ fr(courant.resultat.score, 2) }} ({{ niveau(courant.resultat.score) }}) · calculé en {{ fr(courant.resultat.duree_ms, 0) }} ms</p>
        <p class="small muted">Le score va de 0 (rien de suspect) à 1 (très suspect) ; ce n'est pas une probabilité de cancer. Les zones sont numérotées comme sur l'image.</p>
      </template>
      <template v-else>
        <p>Score du <Sigle t="CNN" /> : <strong>{{ fr(courant.resultat.score, 2) }}</strong>
          <template v-if="courant.resultat.seuil != null">
            · seuil {{ fr(courant.resultat.seuil, 2) }} : <strong>{{ courant.resultat.signale ? 'image signalée' : 'image non signalée' }}</strong>
          </template>
          · calculé en {{ fr(courant.resultat.duree_ms, 0) }} ms</p>
        <p class="small muted">
          Le score va de 0 (bénin) à 1 (malin), tel que le CNN l'a appris sur des seins biopsiés de CMMD ; ce n'est pas une probabilité de cancer.
          Le seuil a été choisi à l'entraînement pour signaler la part visée de seins malins (sensibilité cible) sur les images d'ajustement.
          Les zones, numérotées comme sur l'image, montrent où le CNN a le plus regardé (carte d'attention) : ce ne sont pas des lésions détectées.
        </p>
      </template>
      <ol v-if="courant.resultat.zones.length" class="zones">
        <li v-for="(z, i) in courant.resultat.zones" :key="i"><strong>{{ z.type }}</strong> · {{ courant.resultat.simulation ? 'score' : 'intensité' }} {{ fr(z.score, 2) }}<span class="small muted"> — {{ typesZone[z.type] ?? '' }}</span></li>
      </ol>
      <p v-else class="small muted">Aucune zone signalée.</p>
    </div>

    <details v-if="historique.length" class="historique">
      <summary>Tests précédents sur cette image ({{ historique.length }})</summary>
      <ul>
        <li v-for="t in historique" :key="t.id">
          <button type="button" class="lien" @click="revoir(t)">{{ t.individu }}</button>
          <span class="small muted"> · {{ t.resultat.simulation ? `génération ${t.generation} · simulation` : `CNN ${t.resultat.modele ?? ''}` }} · score {{ fr(t.resultat.score, 2) }} · {{ new Date(t.created_at).toLocaleString('fr-BE') }}</span>
        </li>
      </ul>
    </details>
  </section>
</template>

<style scoped>
.test select { width: 100%; }
.resultat { margin-top: 14px; padding-top: 12px; border-top: 1px solid var(--border); }
.simul { padding: 8px 10px; border-radius: 8px; background: color-mix(in srgb, var(--warn) 16%, transparent); font-size: 0.85rem; }
.simul.reel { background: var(--accent-soft); }
.zones { margin: 6px 0 0; padding-left: 20px; }
.historique { margin-top: 12px; font-size: 0.9rem; }
.historique summary { cursor: pointer; color: var(--accent); }
.historique ul { padding-left: 16px; display: grid; gap: 4px; }
.lien { border: 0; background: none; color: var(--accent); font: inherit; padding: 0; cursor: pointer; text-decoration: underline; }
</style>
