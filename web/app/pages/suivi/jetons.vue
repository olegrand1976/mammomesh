<script setup lang="ts">
// Jetons personnels pour brancher Claude Code sur le serveur MCP.
import type { MCPToken } from '~/types'

definePageMeta({ layout: 'suivi', middleware: 'auth' })
useHead({ title: 'Jetons MCP · MammoMesh' })

const { api } = useApi()
const session = useSessionStore()
const tokens = ref<MCPToken[]>([])
const name = ref('Claude Code')
const scope = ref<'lecture' | 'proposition'>('lecture')
const created = ref('')
const error = ref('')
const origin = computed(() => (import.meta.client ? window.location.origin : ''))

async function load() {
  tokens.value = (await api<{ tokens: MCPToken[] }>('/mcp-tokens')).tokens
}
onMounted(load)

async function create() {
  error.value = ''
  try {
    const res = await api<{ token: string }>('/mcp-tokens', { method: 'POST', body: { name: name.value, scope: scope.value, days: 90 } })
    created.value = res.token
    await load()
  } catch (e) {
    error.value = (e as Error).message
  }
}

async function revoke(id: number) {
  if (!confirm('Révoquer ce jeton ? Claude Code ne pourra plus l\'utiliser.')) return
  await api(`/mcp-tokens/${id}`, { method: 'DELETE' })
  await load()
}

const date = (s: string | null) => (s ? new Date(s).toLocaleDateString('fr-BE') : '—')
const status = (t: MCPToken) => (t.revoked_at ? 'révoqué' : new Date(t.expires_at) < new Date() ? 'expiré' : 'actif')
</script>

<template>
  <div>
    <h1>Jetons MCP</h1>
    <p class="muted">Un jeton relie Claude Code au serveur MCP de MammoMesh avec vos droits. « Lecture » donne les résultats ; « proposition » permet aussi de créer et vérifier des configurations, et de lancer ou régler une campagne si votre rôle le permet. Validité : 90 jours.</p>
    <p v-if="error" class="alert">{{ error }}</p>

    <section class="card">
      <form class="row" @submit.prevent="create">
        <div class="field grow"><label>Nom du jeton</label><input v-model="name" required></div>
        <div class="field"><label>Portée</label>
          <select v-model="scope">
            <option value="lecture">Lecture</option>
            <option value="proposition" :disabled="!session.permissions?.jetons_proposition">Proposition et action</option>
          </select>
        </div>
        <button class="btn btn-primary">Créer</button>
      </form>
      <div v-if="created" class="notice">
        <p><strong>Copiez ce jeton maintenant : il ne sera plus affiché.</strong></p>
        <code class="token">{{ created }}</code>
        <p class="small">Dans le dépôt, exportez <code>MAMMOMESH_MCP_TOKEN</code> avec cette valeur : le fichier <code>.mcp.json</code> s'en sert. Ou, hors du dépôt :</p>
        <code class="token">claude mcp add --transport http mammomesh {{ origin }}/mcp --header "Authorization: Bearer {{ created }}"</code>
      </div>
    </section>

    <section class="card list">
      <table v-if="tokens.length">
        <thead><tr><th>Nom</th><th>Portée</th><th>Créé</th><th>Expire</th><th>Dernier usage</th><th>État</th><th /></tr></thead>
        <tbody>
          <tr v-for="t in tokens" :key="t.id">
            <td>{{ t.name }}</td><td>{{ t.scope }}</td><td>{{ date(t.created_at) }}</td><td>{{ date(t.expires_at) }}</td>
            <td>{{ date(t.last_used_at) }}</td><td>{{ status(t) }}</td>
            <td><button v-if="!t.revoked_at" class="btn btn-danger" @click="revoke(t.id)">Révoquer</button></td>
          </tr>
        </tbody>
      </table>
      <p v-else class="muted">Aucun jeton.</p>
    </section>
  </div>
</template>

<style scoped>
h1 { font-size: 1.4rem; }
.grow { flex: 1; min-width: 200px; }
.row .field { margin-bottom: 0; }
form.row { align-items: flex-end; }
.notice { margin-top: 16px; }
.token { display: block; word-break: break-all; padding: 8px; margin: 6px 0; }
.list { margin-top: 16px; }
</style>
