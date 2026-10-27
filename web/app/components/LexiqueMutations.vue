<script setup lang="ts">
// Lexique des mutations, repliable ; toujours accessible depuis le journal, la fiche et la page « Fonctionnement ».
import { mutations } from '~/utils/mutations'
defineProps<{ ouvert?: boolean }>()
</script>

<template>
  <details class="lexique" :open="ouvert">
    <summary>Lexique des mutations</summary>
    <p class="small muted">Un code dit quel réglage (gène) a changé et dans quel sens : ↑ augmenté, ↓ diminué, + ajouté, − retiré. Survoler un code (ou le sélectionner au clavier) affiche aussi son explication.</p>
    <div class="table-wrap"><table>
      <thead><tr><th>Code</th><th>Réglage (gène)</th><th class="hide-sm">À quoi il sert</th><th>Ce que change la mutation</th></tr></thead>
      <tbody>
        <tr v-for="m in mutations" :key="m.famille">
          <td><code>{{ m.famille }}</code><div class="small muted">{{ m.partie }}</div></td>
          <td>{{ m.gene }}</td>
          <td class="hide-sm small"><TexteGlossaire :texte="m.role" /></td>
          <td class="small"><TexteGlossaire :texte="m.effet" /></td>
        </tr>
      </tbody>
    </table></div>
  </details>
</template>

<style scoped>
.lexique { margin-top: 12px; font-size: 0.9rem; }
.lexique summary { cursor: pointer; color: var(--accent); }
.lexique td { vertical-align: top; }
code { white-space: nowrap; }
</style>
