<script setup lang="ts">
import { fr } from '~/types'
import {
  avancement, duree, estimer, heureFin, libelleDemande, phaseEntrainement, relance, repartir, statutEntrainement, vitesseObservee,
  aideReglages, type DemandeNouveau, type DetailEntrainement, type ReglagesEntrainement, type Releve, type ResumeEntrainement, type VueEntrainements,
} from '~/utils/entrainements'

definePageMeta({ layout: 'suivi', middleware: 'auth' })
useHead({ title: 'Entraînements · MammoMesh' })

const { api } = useApi()
const session = useSessionStore()
const envoi = ref('')
const info = ref('')
const nouveaux = ref<DemandeNouveau[]>([])
const formulaire = ref(false)
const reglages = ref<ReglagesEntrainement | null>(null)
const resolutions = ref<string[]>([])
let defaut: ReglagesEntrainement | null = null
const route = useRoute()
const router = useRouter()
const p = usePalette()

const liste = ref<ResumeEntrainement[]>([])
const configure = ref(true)
const detail = ref<DetailEntrainement | null>(null)
const erreur = ref('')
const groupes = computed(() => repartir(liste.value))
// Onglet : celui de l'adresse, sinon « En cours » s'il y a un entraînement en cours, sinon « Finalisés ».
const vue = computed<VueEntrainements>(() => {
  const q = route.query.vue
  if (q === 'en-cours' || q === 'finalises') return q
  return groupes.value['en-cours'].length || !groupes.value.finalises.length ? 'en-cours' : 'finalises'
})
const affiches = computed(() => groupes.value[vue.value])
const choisi = computed(() => (typeof route.query.nom === 'string' ? route.query.nom : affiches.value[0]?.nom ?? ''))
const onglets: { cle: VueEntrainements; libelle: string }[] = [{ cle: 'en-cours', libelle: 'En cours' }, { cle: 'finalises', libelle: 'Finalisés' }]
let minuteur: ReturnType<typeof setInterval> | undefined
const maintenant = ref(Date.now())
const releves = ref<(Releve & { nom: string })[]>([])

async function charger() {
  try {
    const r = await api<{ configure: boolean; entrainements: ResumeEntrainement[]; nouveaux_en_attente: DemandeNouveau[];
      reglages_defaut: ReglagesEntrainement; resolutions: string[] }>('/entrainements')
    liste.value = r.entrainements
    configure.value = r.configure
    nouveaux.value = r.nouveaux_en_attente ?? []
    defaut = r.reglages_defaut
    resolutions.value = r.resolutions
    detail.value = choisi.value
      ? (await api<{ entrainement: DetailEntrainement }>(`/entrainements/${encodeURIComponent(choisi.value)}`)).entrainement
      : null
    maintenant.value = Date.now()
    const d = detail.value
    if (d) {
      if (releves.value[0] && releves.value[0].nom !== d.nom) releves.value = []
      releves.value = [...releves.value, { nom: d.nom, t: maintenant.value, phase: d.phase, epoque: d.etat?.pas !== undefined ? d.epoque : d.journal.length,
        pas: d.etat?.pas ?? 0, pretraites: d.etat?.pretraites ?? 0 }].slice(-60)
    }
    erreur.value = ''
  } catch (e) {
    erreur.value = (e as Error).message
  }
}

watch(choisi, () => void charger())
onMounted(() => {
  void charger()
  minuteur = setInterval(charger, 5000)
})
onBeforeUnmount(() => clearInterval(minuteur))

const statut = (s: string) => statutEntrainement[s] ?? { libelle: s, classe: '' }
const phase = (s: string) => phaseEntrainement[s] ?? s
const date = (s?: string) => (s ? new Date(s).toLocaleString('fr-BE') : '—')
const pct = computed(() => (detail.value ? avancement(detail.value) : null))
const estimation = computed(() => (detail.value ? estimer(detail.value, maintenant.value, vitesseObservee(releves.value)) : null))
const finListe = (e: ResumeEntrainement) => {
  const r = estimer(e, maintenant.value).resteMax
  return r === null ? '—' : heureFin(maintenant.value, r)
}
function ouvrirFormulaire() {
  reglages.value = defaut ? { ...defaut } : null
  formulaire.value = true
}

async function lancerNouveau() {
  if (!reglages.value) return
  envoi.value = 'nouveau'
  try {
    await api('/entrainements', { method: 'POST', body: { ...reglages.value } })
    info.value = 'Nouvel entraînement demandé : il apparaîtra dans « En cours » et partira dès que le GPU sera libre.'
    formulaire.value = false
    await router.replace({ query: { vue: 'en-cours' } })
    await charger()
  } catch (e) {
    erreur.value = (e as Error).message
  } finally {
    envoi.value = ''
  }
}

async function demander(action: 'relancer' | 'supprimer') {
  const d = detail.value
  if (!d) return
  const texte = action === 'supprimer'
    ? `Supprimer définitivement « ${d.nom} » : modèle, journaux et résultats ?` + (d.statut === 'en_cours' ? '\n\nIl tourne : il sera d\'abord arrêté.' : '')
    : `${relance(d).libelle} « ${d.nom} » ?\n\n${relance(d).explication}`
  if (!window.confirm(texte)) return
  envoi.value = action
  try {
    await api(`/entrainements/${encodeURIComponent(d.nom)}/demandes`, { method: 'POST', body: { action } })
    info.value = action === 'supprimer'
      ? `Suppression de ${d.nom} demandée : le veilleur s'en charge dans quelques secondes.`
      : `${relance(d).libelle} ${d.nom} : demande envoyée au veilleur.`
    if (action === 'supprimer') await router.replace({ query: { vue: vue.value } })
    await charger()
  } catch (e) {
    erreur.value = (e as Error).message
  } finally {
    envoi.value = ''
  }
}

const quand = (reste: number) => `dans ${duree(reste)} (vers ${heureFin(maintenant.value, reste)})`

const option = computed(() => {
  const j = detail.value?.journal ?? []
  const axe = { axisLine: { lineStyle: { color: p.value.grid } }, axisLabel: { color: p.value.muted }, nameTextStyle: { color: p.value.muted } }
  return {
    animation: false,
    textStyle: { color: p.value.text, fontFamily: 'inherit' },
    grid: { left: 52, right: 56, top: 48, bottom: 36 },
    legend: { top: 0, textStyle: { color: p.value.muted } },
    tooltip: { trigger: 'axis', valueFormatter: (v: number | null) => (v === null || v === undefined ? '—' : v.toFixed(3)) },
    xAxis: { type: 'category', name: 'époque', nameLocation: 'middle', nameGap: 24, data: j.map(e => e.epoque), ...axe },
    yAxis: [
      { type: 'value', name: 'AUC (ajustement)', min: 0.5, max: 1, splitLine: { lineStyle: { color: p.value.grid } }, ...axe },
      { type: 'value', name: 'perte', position: 'right', scale: true, splitLine: { show: false }, ...axe },
    ],
    series: [
      { name: 'AUC par sein', type: 'line', data: j.map(e => e.auc_sein), lineStyle: { width: 2, color: p.value.accent }, itemStyle: { color: p.value.accent } },
      { name: 'AUC par image', type: 'line', data: j.map(e => e.auc_image), lineStyle: { width: 1.5, color: p.value.muted }, itemStyle: { color: p.value.muted } },
      { name: 'Perte (entraînement)', type: 'line', yAxisIndex: 1, data: j.map(e => e.perte), lineStyle: { width: 1.5, type: 'dotted', color: p.value.second }, itemStyle: { color: p.value.second } },
    ],
  }
})
</script>

<template>
  <div>
    <div class="row head">
      <h1>Entraînements du CNN</h1>
      <div class="spacer" />
      <button v-if="session.permissions?.gerer_entrainements && configure" class="btn btn-primary" :disabled="formulaire" @click="ouvrirFormulaire">
        Nouvel entraînement
      </button>
    </div>
    <p class="muted small intro">
      Suivi des entraînements lancés avec <code>python -m mammomesh_engine.cnn</code>, rafraîchi toutes les 5 secondes.
      Les mesures se font sur l'ensemble d'ajustement (E-ajust) ; la validation (E-val) n'est mesurée qu'à la fin, sur des patientes jamais vues.
    </p>
    <details class="card cnn">
      <summary>Qu'est-ce qu'un CNN, et que montre cette page ?</summary>
      <p>
        Un <strong>CNN</strong> (<em>convolutional neural network</em>, réseau de neurones convolutif) est un programme qui apprend
        à reconnaître des motifs dans des images. Ses premières couches repèrent des bords et des textures, les suivantes des
        formes de plus en plus complexes. On ne lui écrit pas de règles : on lui montre des milliers d'exemples dont on connaît
        la réponse, et il ajuste ses millions de paramètres pour se tromper de moins en moins.
      </p>
      <p>
        Dans MammoMesh, le CNN est le <strong>regard</strong> : il lit la mammographie et donne un score ; CephaloMesh, ensuite,
        <strong>raisonne</strong> sur ce qu'il a vu (concordance des vues, symétrie, évolution depuis l'examen précédent).
        Le CNN entraîné ici (EfficientNet-B0, partant de poids appris sur des photos ordinaires) apprend, sur la base CMMD,
        à dire si un sein qui porte une anomalie est plutôt <strong>malin ou bénin</strong> (prouvé par biopsie).
      </p>
      <ul>
        <li><strong>Époque</strong> : un passage complet sur toutes les images d'entraînement.</li>
        <li><strong>Perte</strong> : l'erreur moyenne pendant l'apprentissage ; elle doit baisser.</li>
        <li><strong>AUC</strong> : de 0,5 (hasard) à 1 (parfait), la probabilité qu'un sein malin reçoive un score plus haut qu'un sein bénin.</li>
        <li><strong>Ajustement (E-ajust)</strong> : patientes mises de côté pour suivre les progrès et choisir le seuil ;
          <strong>validation (E-val)</strong> : patientes jamais vues, mesurées une seule fois à la fin.</li>
      </ul>
      <p class="small muted">Outil de recherche : aucun résultat ne sert à un diagnostic.</p>
    </details>
    <p v-if="erreur" class="alert">{{ erreur }}</p>
    <p v-if="info" class="attente">{{ info }}</p>
    <p v-if="!configure" class="alert">
      Le suivi n'est pas configuré : l'API doit connaître le dossier des modèles (variable <code>MODELES_DIR</code> ;
      avec Docker, le dossier <code>./modeles</code> est monté d'office).
    </p>

    <form v-if="formulaire && reglages" class="card nouveau" @submit.prevent="lancerNouveau">
      <h2>Nouvel entraînement</h2>
      <p class="small muted">
        Sur la banque CMMD du serveur. La demande va au veilleur (service mammomesh-entrainement), qui lance l'entraînement
        dès que le GPU est libre ; il apparaît aussitôt dans « En cours ». Les valeurs proposées sont celles par défaut.
      </p>
      <div class="grid champs">
        <label v-for="cle in (['epoques', 'patience', 'limite', 'lr', 'sensibilite_cible', 'graine'] as const)" :key="cle">
          <span>{{ aideReglages[cle].libelle }}</span>
          <input v-model.number="reglages[cle]" type="number" required
            :step="cle === 'lr' ? 0.00001 : cle === 'sensibilite_cible' ? 0.01 : 1" :min="cle === 'lr' ? 0.00001 : 0">
          <small class="muted">{{ aideReglages[cle].aide }}</small>
        </label>
        <label>
          <span>{{ aideReglages.resolution.libelle }}</span>
          <select v-model="reglages.resolution"><option v-for="r in resolutions" :key="r" :value="r">{{ r.replace('x', ' × ') }}</option></select>
          <small class="muted">{{ aideReglages.resolution.aide }}</small>
        </label>
        <label class="case">
          <span><input v-model="reglages.sans_preentrainement" type="checkbox"> {{ aideReglages.sans_preentrainement.libelle }}</span>
          <small class="muted">{{ aideReglages.sans_preentrainement.aide }}</small>
        </label>
      </div>
      <div class="row actions">
        <button class="btn btn-primary" type="submit" :disabled="!!envoi">{{ envoi === 'nouveau' ? 'Envoi…' : 'Lancer' }}</button>
        <button class="btn" type="button" @click="formulaire = false">Annuler</button>
      </div>
    </form>

    <nav class="onglets" aria-label="Entraînements">
      <NuxtLink v-for="o in onglets" :key="o.cle" :to="{ query: { vue: o.cle } }" :class="{ actif: vue === o.cle }"
        :aria-current="vue === o.cle ? 'page' : undefined">
        {{ o.libelle }} <span class="compte">{{ groupes[o.cle].length + (o.cle === 'en-cours' ? nouveaux.length : 0) }}</span>
      </NuxtLink>
    </nav>

    <section class="card">
      <p v-for="n in (vue === 'en-cours' ? nouveaux : [])" :key="n.date" class="attente">
        Nouvel entraînement demandé par {{ n.par }} le {{ date(n.date) }} ({{ n.reglages.epoques }} époques, {{ n.reglages.resolution.replace('x', ' × ') }}) :
        le veilleur le crée dans quelques secondes. S'il ne tourne pas, rien ne se passera.
      </p>
      <div v-if="affiches.length" class="table-wrap"><table>
        <thead><tr><th>Entraînement</th><th>État</th><th>Phase</th><th>Époque</th><th>Meilleure AUC (ajustement)</th><th class="hide-sm">AUC validation</th><th v-if="vue === 'en-cours'">Fin estimée</th><th class="hide-sm">{{ vue === 'finalises' ? 'Fin' : 'Dernière nouvelle' }}</th></tr></thead>
        <tbody>
          <tr v-for="e in affiches" :key="e.nom" :class="{ choisi: e.nom === detail?.nom }">
            <td><a href="#" @click.prevent="router.replace({ query: { vue, nom: e.nom } })">{{ e.nom }}</a></td>
            <td>
              <span class="badge" :class="statut(e.statut).classe">{{ statut(e.statut).libelle }}</span>
              <div v-if="e.demande" class="small muted">{{ libelleDemande[e.demande.action] }}…</div>
            </td>
            <td>{{ phase(e.phase) }}</td>
            <td>{{ e.epoque }}<template v-if="e.epoques"> / {{ e.epoques }}</template></td>
            <td>{{ fr(e.meilleure_auc_sein) }}</td>
            <td class="hide-sm">{{ fr(e.auc_sein_validation) }}</td>
            <td v-if="vue === 'en-cours'" class="small" title="Au plus tard, si toutes les époques prévues sont faites">{{ finListe(e) }}</td>
            <td class="small hide-sm">{{ date(e.maj) }}</td>
          </tr>
        </tbody>
      </table></div>
      <p v-else-if="configure && !(vue === 'en-cours' && nouveaux.length)" class="muted">{{ vue === 'en-cours' ? 'Aucun entraînement en cours.' : 'Aucun entraînement finalisé pour l\'instant.' }}</p>
    </section>

    <section v-if="detail" class="card">
      <div class="row titre">
        <h2>{{ detail.nom }}</h2>
        <span class="badge" :class="statut(detail.statut).classe">{{ statut(detail.statut).libelle }}</span>
        <div class="spacer" />
        <template v-if="session.permissions?.gerer_entrainements">
          <button class="btn" :disabled="!relance(detail).possible || !!detail.demande || !!envoi" :title="relance(detail).explication"
            @click="demander('relancer')">{{ envoi === 'relancer' ? 'Envoi…' : relance(detail).libelle }}</button>
          <button class="btn btn-danger" :disabled="!!detail.demande || !!envoi" title="Effacer le modèle, les journaux et les résultats"
            @click="demander('supprimer')">{{ envoi === 'supprimer' ? 'Envoi…' : 'Supprimer' }}</button>
        </template>
      </div>
      <p v-if="detail.demande" class="attente">
        {{ libelleDemande[detail.demande.action] }} par {{ detail.demande.par }} le {{ date(detail.demande.date) }} :
        le veilleur (service mammomesh-entrainement) s'en charge dans quelques secondes. S'il ne tourne pas, rien ne se passera.
      </p>
      <p class="phase">
        <strong>{{ phase(detail.phase) }}</strong><template v-if="detail.etape"> · {{ detail.etape }}</template>
        <template v-if="detail.epoques"> · époque {{ detail.epoque }} / {{ detail.epoques }}</template>
        <template v-if="detail.etat?.pas_par_epoque && detail.phase === 'entrainement'"> · pas {{ detail.etat.pas ?? 0 }} / {{ detail.etat.pas_par_epoque }}</template>
        <template v-if="detail.phase === 'pretraitement' && detail.etat?.a_pretraiter"> · {{ detail.etat.pretraites ?? 0 }} / {{ detail.etat.a_pretraiter }} images</template>
      </p>
      <div v-if="pct !== null" class="barre" role="progressbar" :aria-valuenow="Math.round(pct * 100)" aria-valuemin="0" aria-valuemax="100">
        <div :style="{ width: `${(pct * 100).toFixed(1)}%` }" />
      </div>
      <p class="small muted">Lancé le {{ date(detail.debut) }} · dernière nouvelle {{ date(detail.maj) }}</p>
      <div v-if="estimation && detail.statut === 'en_cours'" class="temps">
        <h3>Temps estimés</h3>
        <ul>
          <li v-if="estimation.restePretraitement !== null"><strong>Prétraitement</strong> : fin {{ quand(estimation.restePretraitement) }}</li>
          <li v-if="estimation.resteEpoque !== null">
            <strong>Époque {{ estimation.epoqueCourante }} / {{ detail.epoques }}</strong> : fin {{ quand(estimation.resteEpoque) }}
            <template v-if="estimation.dureeEpoque !== null"> · une époque dure environ {{ duree(estimation.dureeEpoque) }}</template>
          </li>
          <li v-if="estimation.resteMax !== null">
            <strong>Fin de l'entraînement</strong> : au plus tard {{ quand(estimation.resteMax) }}, si les {{ detail.epoques }} époques sont faites
          </li>
          <li v-if="estimation.resteMin !== null">
            au plus tôt {{ quand(estimation.resteMin) }}, si l'AUC ne progresse plus d'ici l'époque {{ estimation.epoqueArret }} (arrêt anticipé)
          </li>
          <li v-if="estimation.resteEpoque === null && estimation.restePretraitement === null" class="muted">
            Pas encore d'estimation : il faut quelques pas d'apprentissage ({{ phase(detail.phase).toLowerCase() }} en cours).
          </li>
        </ul>
        <p class="small muted">
          Estimations recalculées à chaque rafraîchissement, d'après la vitesse mesurée ; la première époque, qui compile le modèle,
          est plus longue que les suivantes. L'évaluation finale (ajustement et validation) est comprise.
        </p>
      </div>
      <p v-if="detail.statut === 'sans_nouvelles'" class="alert">
        Aucune nouvelle depuis longtemps : l'entraînement s'est peut-être arrêté sans le signaler (machine éteinte, processus tué).
      </p>
      <p v-if="detail.message" :class="detail.statut === 'en_attente_gpu' ? 'attente' : 'alert'">{{ detail.message }}</p>
      <p v-if="detail.statut === 'en_attente_gpu'" class="small muted">
        Le veilleur (service mammomesh-entrainement) reprendra cet entraînement à sa dernière époque finie dès que le GPU sera libre.
      </p>
      <p v-if="!detail.avec_etat && detail.statut === 'en_cours'" class="small muted">
        Cet entraînement a été lancé avant le suivi détaillé : il n'est mis à jour qu'à la fin de chaque époque.
      </p>

      <EChart v-if="detail.journal.length" class="graphique" :option="option" />
      <p v-else class="muted small">Pas encore d'époque terminée.</p>

      <div v-if="detail.rapport?.validation" class="resultats">
        <h3>Résultat (validation, patientes jamais vues)</h3>
        <div class="grid chiffres">
          <div><div class="small muted">AUC par sein</div><div class="valeur">{{ fr(detail.rapport.validation.auc_sein) }}</div></div>
          <div><div class="small muted">AUC par image</div><div class="valeur">{{ fr(detail.rapport.validation.auc_image) }}</div></div>
          <div><div class="small muted">Sensibilité au seuil {{ fr(detail.rapport.validation.seuil, 2) }}</div><div class="valeur">{{ fr(detail.rapport.validation.sensibilite_sein) }}</div></div>
          <div><div class="small muted">Spécificité</div><div class="valeur">{{ fr(detail.rapport.validation.specificite_sein) }}</div></div>
        </div>
        <p class="small muted">
          {{ detail.rapport.validation.seins }} seins dont {{ detail.rapport.validation.seins_malins }} malins.
          Le seuil est choisi sur l'ajustement pour la sensibilité cible.
          <template v-if="detail.rapport.tache">Tâche : {{ detail.rapport.tache }}.</template>
        </p>
      </div>

      <details v-if="detail.etat?.plan?.length" class="plan">
        <summary>Plan matériel</summary>
        <pre>{{ detail.etat.plan.join('\n') }}</pre>
        <p v-if="detail.etat.commande" class="small muted">Commande : <code>{{ detail.etat.commande }}</code></p>
      </details>
    </section>
  </div>
</template>

<style scoped>
.head { margin-bottom: 8px; }
.head h1 { margin: 0; font-size: 1.4rem; }
.intro { margin-top: 0; }
.titre { gap: 10px; }
.titre h2 { margin: 0; }
.phase { margin: 8px 0; }
.barre { height: 8px; background: var(--surface-2); border-radius: 999px; overflow: hidden; margin: 6px 0; }
.barre > div { height: 100%; background: var(--accent); transition: width 0.4s; }
.graphique { height: 320px; width: 100%; margin-top: 12px; }
.resultats h3 { font-size: 1rem; margin: 16px 0 8px; }
.chiffres { grid-template-columns: repeat(auto-fit, minmax(150px, 1fr)); gap: 12px; }
.valeur { font-size: 1.3rem; font-weight: 600; }
.plan pre { white-space: pre-wrap; font-size: 0.8rem; }
tr.choisi td { background: var(--surface-2); }
.nouveau h2 { margin-top: 0; }
.champs { grid-template-columns: repeat(auto-fit, minmax(240px, 1fr)); gap: 14px; margin: 12px 0; }
.champs label { display: grid; gap: 4px; }
.champs label > span { font-weight: 600; font-size: 0.9rem; }
.champs .case > span { font-weight: 500; }
.actions { gap: 8px; }
.cnn summary { cursor: pointer; font-weight: 600; }
.cnn p, .cnn ul { margin: 10px 0 0; }
.cnn ul { padding-left: 20px; display: grid; gap: 4px; }
.attente { background: var(--accent-soft); color: var(--accent); padding: 8px 12px; border-radius: 8px; }
.temps h3 { font-size: 1rem; margin: 12px 0 6px; }
.temps ul { padding-left: 20px; display: grid; gap: 4px; margin: 0 0 6px; }
section + section { margin-top: 16px; }
.onglets { display: flex; gap: 6px; margin: 12px 0; border-bottom: 1px solid var(--border); }
.onglets a { padding: 6px 12px; color: var(--muted); border-bottom: 2px solid transparent; margin-bottom: -1px; }
.onglets a.actif { color: var(--text); border-bottom-color: var(--accent); text-decoration: none; }
.compte { font-size: 0.75rem; background: var(--surface-2); border-radius: 999px; padding: 0 7px; margin-left: 4px; }
</style>
