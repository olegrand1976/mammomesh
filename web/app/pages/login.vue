<script setup lang="ts">
definePageMeta({ layout: false })
useHead({ title: 'Connexion · MammoMesh' })

const session = useSessionStore()
const username = ref('')
const password = ref('')
const error = ref('')
const busy = ref(false)

onMounted(async () => {
  if (await session.refresh()) await navigateTo('/suivi')
})

async function submit() {
  error.value = ''
  busy.value = true
  try {
    await session.login(username.value, password.value)
    await navigateTo('/suivi')
  } catch (e) {
    error.value = (e as Error).message
  } finally {
    busy.value = false
  }
}
</script>

<template>
  <main class="login">
    <form class="card" @submit.prevent="submit">
      <NuxtLink to="/" class="brand"><img src="/favicon.svg" alt="" width="28" height="28"> MammoMesh</NuxtLink>
      <p class="muted small">Interface de suivi · accès sur le réseau local</p>
      <p v-if="error" class="alert" role="alert">{{ error }}</p>
      <div class="field">
        <label for="u">Identifiant</label>
        <input id="u" v-model="username" autocomplete="username" required autofocus>
      </div>
      <div class="field">
        <label for="p">Mot de passe</label>
        <input id="p" v-model="password" type="password" autocomplete="current-password" required>
      </div>
      <button class="btn btn-primary full" :disabled="busy">{{ busy ? 'Connexion…' : 'Se connecter' }}</button>
      <p class="small muted note">Comptes créés par l'administrateur. Outil de recherche, sans usage clinique.</p>
    </form>
  </main>
</template>

<style scoped>
.login { min-height: 100vh; display: grid; place-items: center; padding: 16px; }
form { width: 100%; max-width: 380px; padding: 28px; }
.brand { display: inline-flex; align-items: center; gap: 10px; font-weight: 600; font-size: 1.25rem; color: var(--text); margin-bottom: 4px; }
.brand:hover { text-decoration: none; }
.full { width: 100%; justify-content: center; margin-top: 6px; }
.note { margin: 16px 0 0; }
</style>
