<script setup lang="ts">
import { roleLabel } from '~/types'

definePageMeta({ layout: 'suivi', middleware: 'auth' })
useHead({ title: 'Mon compte · MammoMesh' })

const session = useSessionStore()
const actuel = ref('')
const nouveau = ref('')
const confirmation = ref('')
const error = ref('')
const done = ref(false)
const busy = ref(false)

const tooShort = computed(() => nouveau.value.length > 0 && [...nouveau.value].length < 12)
const mismatch = computed(() => confirmation.value.length > 0 && confirmation.value !== nouveau.value)

async function submit() {
  error.value = ''
  done.value = false
  if (tooShort.value || mismatch.value) return
  busy.value = true
  const forced = session.user?.must_change_password
  try {
    await session.changePassword(actuel.value, nouveau.value)
    actuel.value = nouveau.value = confirmation.value = ''
    done.value = true
    if (forced) await navigateTo('/suivi')
  } catch (e) {
    error.value = (e as Error).message
  } finally {
    busy.value = false
  }
}
</script>

<template>
  <div class="narrow">
    <h1>Mon compte</h1>
    <p v-if="session.user" class="muted">{{ session.user.username }} · {{ roleLabel[session.user.role] }}</p>
    <p v-if="session.user?.must_change_password" class="notice" role="status">
      Ce compte utilise encore le mot de passe par défaut. Choisissez-en un nouveau pour accéder au suivi.
    </p>
    <form class="card" @submit.prevent="submit">
      <h2>Changer le mot de passe</h2>
      <p v-if="error" class="alert" role="alert">{{ error }}</p>
      <p v-if="done" class="notice" role="status">Mot de passe changé. Les autres sessions de ce compte sont fermées.</p>
      <div class="field">
        <label for="actuel">Mot de passe actuel</label>
        <input id="actuel" v-model="actuel" type="password" autocomplete="current-password" required>
      </div>
      <div class="field">
        <label for="nouveau">Nouveau mot de passe (12 caractères au moins)</label>
        <input id="nouveau" v-model="nouveau" type="password" autocomplete="new-password" required :aria-invalid="tooShort">
        <span v-if="tooShort" class="small bad">Encore {{ 12 - [...nouveau].length }} caractère(s).</span>
      </div>
      <div class="field">
        <label for="confirmation">Confirmation</label>
        <input id="confirmation" v-model="confirmation" type="password" autocomplete="new-password" required :aria-invalid="mismatch">
        <span v-if="mismatch" class="small bad">Les deux saisies diffèrent.</span>
      </div>
      <button class="btn btn-primary" :disabled="busy || tooShort || mismatch">{{ busy ? 'Enregistrement…' : 'Enregistrer' }}</button>
    </form>
  </div>
</template>

<style scoped>
.narrow { max-width: 480px; }
h1 { font-size: 1.4rem; margin-bottom: 4px; }
.bad { color: var(--bad); }
</style>
