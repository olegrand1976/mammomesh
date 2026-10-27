<script setup lang="ts">
// Icônes de la barre latérale : traits simples, couleur du texte (clair et sombre).
const props = defineProps<{ nom: string }>()
const cercle = (cx: number, cy: number, r: number) => `M${cx - r} ${cy}a${r} ${r} 0 1 0 ${2 * r} 0a${r} ${r} 0 1 0 ${-2 * r} 0`
const traces: Record<string, string[]> = {
  accueil: ['M3 11l9-7 9 7v9a1 1 0 0 1-1 1h-5v-6H9v6H4a1 1 0 0 1-1-1z'],
  entrainement: ['M7 6h10a1 1 0 0 1 1 1v10a1 1 0 0 1-1 1H7a1 1 0 0 1-1-1V7a1 1 0 0 1 1-1z', 'M10 10h4v4h-4z',
    'M9 3v3M15 3v3M9 18v3M15 18v3M3 9h3M3 15h3M18 9h3M18 15h3'],
  campagnes: ['M3 3v18h18', 'M7 15l4-4 3 3 5-6'],
  ajouter: [cercle(12, 12, 9), 'M12 8v8M8 12h8'],
  image: ['M4 4h16a1 1 0 0 1 1 1v14a1 1 0 0 1-1 1H4a1 1 0 0 1-1-1V5a1 1 0 0 1 1-1z', cercle(9, 10, 2), 'M21 16l-5-5-9 9'],
  livre: ['M4 19V5a2 2 0 0 1 2-2h13v16H6a2 2 0 0 0-2 2 2 2 0 0 0 2 2h13'],
  cle: [cercle(8, 15, 4), 'M11 12l9-9M16 7l3 3'],
  compte: [cercle(12, 8, 4), 'M4 21a8 8 0 0 1 16 0'],
  sortie: ['M15 4h4a1 1 0 0 1 1 1v14a1 1 0 0 1-1 1h-4', 'M10 17l-5-5 5-5M5 12h11'],
  replier: ['M11 17l-5-5 5-5M18 17l-5-5 5-5'],
  deplier: ['M13 7l5 5-5 5M6 7l5 5-5 5'],
  menu: ['M4 6h16M4 12h16M4 18h16'],
  fermer: ['M6 6l12 12M18 6L6 18'],
}
</script>

<template>
  <svg class="icone" viewBox="0 0 24 24" width="20" height="20" fill="none" stroke="currentColor" stroke-width="1.8"
    stroke-linecap="round" stroke-linejoin="round" aria-hidden="true" focusable="false">
    <path v-for="d in traces[props.nom] ?? []" :key="d" :d="d" />
  </svg>
</template>

<style scoped>
.icone { flex: none; }
</style>
