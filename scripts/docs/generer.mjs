// Documents du dépôt tirés des données de l'interface (web/app/utils) : docs/glossaire.md,
// docs/installation-windows-11.md et le JSON qui sert au PDF. Lancé par scripts/generer-docs.sh.
import { writeFileSync } from 'node:fs'
import { glossaire } from '../../web/app/utils/glossaire.ts'
import { etatWindows, partiesWindows } from '../../web/app/utils/installationWindows.ts'
const md = s => s.replaceAll('\\', '\\\\')   // barres obliques inverses de Windows, gardées telles quelles

// Glossaire
const G = ['# Glossaire', '', 'Dans l\'application : menu « Documentation » › « Glossaire » (`/suivi/documentation/glossaire`). Ce document en reprend le contenu. Dans chaque écran, un sigle souligné en pointillé affiche cette explication au survol ou au clavier.', '']
for (const [titre, sigle] of [['Sigles', true], ['Mots', false]]) {
  G.push(`## ${titre}`, '')
  for (const t of glossaire.filter(x => x.sigle === sigle).sort((a, b) => a.terme.localeCompare(b.terme, 'fr'))) {
    G.push(`**${t.terme}**${t.forme ? ` (${t.forme})` : ''} : ${t.definition}`, '')
  }
}
writeFileSync(process.argv[2] + '/glossaire.md', G.join('\n'))

// Windows 11
const W = ['# Installer MammoMesh sur Windows 11', '', 'Dans l\'application : menu « Documentation » › « Installation sur Windows 11 » (`/suivi/documentation/installation-windows`), avec le PDF à télécharger. Version PDF dans le dépôt : [installation-windows-11.pdf](installation-windows-11.pdf). Installation sur un serveur Linux : [installation-serveur.md](installation-serveur.md).', '', `> ${etatWindows}.`, '']
for (const p of partiesWindows) {
  W.push(`## ${p.titre}`, '')
  for (const b of p.blocs) {
    if (b.p) W.push(md(b.p), '')
    if (b.etapes) { b.etapes.forEach((e, i) => W.push(`${i + 1}. ${md(e)}`)); W.push('') }
    if (b.points) { b.points.forEach(e => W.push(`- ${md(e)}`)); W.push('') }
    if (b.table) {
      W.push(`| ${b.table.entetes.join(' | ')} |`, `|${b.table.entetes.map(() => '---').join('|')}|`)
      for (const l of b.table.lignes) W.push(`| ${l.map(md).join(' | ')} |`)
      W.push('')
    }
    if (b.code) W.push('```' + (b.langage ?? ''), b.code, '```', '')
  }
}
writeFileSync(process.argv[2] + '/installation-windows-11.md', W.join('\n'))
if (process.argv[3]) writeFileSync(process.argv[3], JSON.stringify({ etat: etatWindows, parties: partiesWindows }, null, 1))
