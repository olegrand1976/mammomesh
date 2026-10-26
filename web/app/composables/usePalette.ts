// Couleurs des graphiques lues dans les variables CSS : clair et sombre sans duplication.
export function usePalette() {
  const palette = ref({ text: '#1d1d1b', muted: '#66645e', grid: '#e2e0db', accent: '#2463a6', second: '#a8661b', faint: '#c9c7c1' })

  function read() {
    if (!import.meta.client) return
    const s = getComputedStyle(document.documentElement)
    const v = (n: string) => s.getPropertyValue(n).trim()
    palette.value = { text: v('--text'), muted: v('--muted'), grid: v('--border'), accent: v('--accent'), second: v('--warn'), faint: v('--border') }
  }

  onMounted(() => {
    read()
    const mq = window.matchMedia('(prefers-color-scheme: dark)')
    mq.addEventListener('change', read)
    onBeforeUnmount(() => mq.removeEventListener('change', read))
  })
  return palette
}
