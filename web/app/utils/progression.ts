// Lecture, en mots simples, de l'évolution d'un entraînement du CNN (page Entraînements) :
// ce qu'a gagné l'AUC depuis le début, à la dernière époque, et ce que la courbe laisse prévoir.
// Sert aussi à vérifier les exemples du guide (un test).
import { duree, type EpoqueEntrainement, type RapportEntrainement } from '~/utils/entrainements'

export type Verdict = 'debut' | 'progresse' | 'ralentit' | 'plateau' | 'surapprentissage' | 'stagne'

export interface Progression {
  verdict: Verdict
  titre: string
  premiere: number
  meilleure: number
  epoqueMeilleure: number
  derniere: number
  gainTotal: number // meilleure − première
  gainDerniere: number | null // dernière − avant-dernière
  depuisMeilleure: number // époques terminées depuis le record
  phrases: string[]
}

export const titres: Record<Verdict, string> = {
  debut: 'Trop tôt pour juger',
  progresse: 'Le CNN progresse',
  ralentit: 'Les progrès ralentissent',
  plateau: 'Plus de progrès pour l\'instant',
  surapprentissage: 'Le CNN commence à apprendre par cœur',
  stagne: 'Le CNN n\'apprend presque rien',
}

const fr = (x: number, d = 3) => x.toLocaleString('fr-BE', { minimumFractionDigits: d, maximumFractionDigits: d })
const signe = (x: number) => (x >= 0 ? '+' : '−') + fr(Math.abs(x))
const epoques = (n: number) => `${n} époque${n > 1 ? 's' : ''}`

/** « Sur 100 paires… » : traduction concrète d'une AUC. */
export function surCent(auc: number): string {
  return `sur 100 paires (un sein malin, un sein bénin) tirées au hasard, il désigne le bon environ ${Math.round(auc * 100)} fois`
}

/** Ce que représente la perte, et comment elle a évolué (courbe pointillée, axe de droite). */
export function lirePerte(journal: EpoqueEntrainement[]): string[] {
  const p = journal.map(e => e.perte).filter((x): x is number => typeof x === 'number' && Number.isFinite(x))
  if (!p.length) return []
  const out = ['La perte (pointillés, axe de droite) mesure à quel point le CNN se trompe sur les images d\'apprentissage : elle baisse quand ses réponses se rapprochent du bon diagnostic et qu\'il en est plus sûr. Elle se calcule sur les images qu\'il apprend, l\'AUC sur les images d\'ajustement qu\'il n\'apprend pas : les deux courbes peuvent donc diverger.']
  if (p.length > 1) {
    const a = p[0]!, b = p[p.length - 1]!
    const pct = a ? Math.round(Math.abs(b - a) / a * 1000) / 10 : 0
    const sens = b < a ? 'baissé' : b > a ? 'monté' : 'pas bougé'
    out.push(`Ici, elle a ${sens}${b !== a ? ` de ${fr(a)} à ${fr(b)} (${b < a ? '−' : '+'}${pct.toLocaleString('fr-BE')} %)` : ` (${fr(a)})`}. L'axe de droite est zoomé sur les valeurs atteintes : un petit écart y paraît grand.`)
  }
  return out
}

/** Lecture de la courbe d'AUC par sein (ajustement) et de la perte, époque par époque. */
export function lireProgression(journal: EpoqueEntrainement[], patience = 5): Progression | null {
  const j = journal.filter(e => e.auc_sein !== null)
  if (!j.length) return null
  const auc = j.map(e => e.auc_sein!)
  const premiere = auc[0]!
  const derniere = auc[auc.length - 1]!
  let iMax = 0
  auc.forEach((a, i) => { if (a > auc[iMax]!) iMax = i })
  const meilleure = auc[iMax]!
  const epoqueMeilleure = j[iMax]!.epoque
  const depuisMeilleure = j.length - 1 - iMax
  const gainTotal = meilleure - premiere
  const gainDerniere = j.length > 1 ? derniere - auc[auc.length - 2]! : null
  const perteBaisse = j.length > 1 && j[j.length - 1]!.perte < j[iMax]!.perte

  let verdict: Verdict
  if (j.length < 2) verdict = 'debut'
  else if (j.length >= 4 && gainTotal < 0.02) verdict = 'stagne'
  else if (depuisMeilleure >= 2 && perteBaisse && derniere < meilleure - 0.01) verdict = 'surapprentissage'
  else if (depuisMeilleure >= 1) verdict = 'plateau'
  else if ((gainDerniere ?? 0) >= 0.01) verdict = 'progresse'
  else verdict = 'ralentit'

  const phrases: string[] = []
  phrases.push(`AUC par sein à la première époque : ${fr(premiere)} ; meilleure : ${fr(meilleure)} (époque ${epoqueMeilleure}), soit ${signe(gainTotal)} depuis le début. Concrètement, ${surCent(meilleure)}.`)
  phrases.push(...lirePerte(journal))
  if (verdict === 'debut') phrases.push('Une seule époque terminée : il faut au moins deux points pour voir une tendance.')
  if (verdict === 'progresse') phrases.push(`Dernière époque : ${signe(gainDerniere!)}, un nouveau record. L'apprentissage porte encore ses fruits.`)
  if (verdict === 'ralentit') phrases.push(`Dernière époque : ${signe(gainDerniere!)}. C'est encore un record, mais le gain devient petit : le CNN a déjà appris l'essentiel de ce que ces images peuvent lui apprendre.`)
  if (verdict === 'plateau' || verdict === 'surapprentissage' || verdict === 'stagne') {
    const reste = patience - depuisMeilleure
    phrases.push(`Pas de nouveau record depuis ${epoques(depuisMeilleure)}. ${reste > 0
      ? `Si cela dure encore ${epoques(reste)}, l'entraînement s'arrêtera de lui-même (arrêt anticipé).`
      : 'C\'est le moment où l\'arrêt anticipé intervient.'} Quoi qu'il arrive, c'est la version de l'époque ${epoqueMeilleure} qui sera gardée.`)
  }
  if (verdict === 'surapprentissage') phrases.push('La perte continue de baisser alors que l\'AUC recule : le CNN s\'améliore sur les images qu\'il connaît, mais moins bien sur les autres. Il commence à les apprendre par cœur.')
  if (verdict === 'stagne') phrases.push('L\'AUC reste proche de 0,5, le niveau du hasard. Causes possibles : trop peu d\'images, un réglage d\'apprentissage inadapté, ou un problème dans les données.')
  return { verdict, titre: titres[verdict], premiere, meilleure, epoqueMeilleure, derniere, gainTotal, gainDerniere, depuisMeilleure, phrases }
}

/** Trois scénarios types, chiffres illustratifs, pour apprendre à lire la courbe. */
export interface Exemple { titre: string; attendu: Verdict; journal: EpoqueEntrainement[]; lecture: string }

const serie = (auc: number[], perte: number[]): EpoqueEntrainement[] =>
  auc.map((a, i) => ({ epoque: i + 1, perte: perte[i]!, auc_image: null, auc_sein: a, duree_s: 0 }))

export const exemples: Exemple[] = [
  {
    titre: 'Un apprentissage qui se passe bien',
    attendu: 'ralentit',
    journal: serie([0.64, 0.68, 0.72, 0.75, 0.77, 0.78, 0.785, 0.787], [0.66, 0.62, 0.58, 0.55, 0.53, 0.52, 0.51, 0.505]),
    lecture: 'Gros progrès au début (+0,04 par époque), puis de plus en plus petits (+0,002 à la fin). La perte baisse en même temps. Le CNN a appris l\'essentiel : continuer coûterait des heures pour presque rien. On passe de 64 à 79 paires bien classées sur 100.',
  },
  {
    titre: 'Le CNN apprend par cœur (surapprentissage)',
    attendu: 'surapprentissage',
    journal: serie([0.66, 0.71, 0.74, 0.76, 0.75, 0.74, 0.73, 0.72], [0.64, 0.58, 0.52, 0.46, 0.40, 0.34, 0.29, 0.25]),
    lecture: 'Jusqu\'à l\'époque 4, tout va bien. Ensuite, la perte continue de chuter mais l\'AUC recule : comme un élève qui retient les réponses du livre sans comprendre, le CNN réussit de mieux en mieux les images qu\'il a vues et de moins en moins les autres. C\'est la version de l\'époque 4 qui est gardée, et l\'arrêt anticipé coupera l\'entraînement à l\'époque 9.',
  },
  {
    titre: 'Rien n\'est appris',
    attendu: 'stagne',
    journal: serie([0.51, 0.525, 0.50, 0.52, 0.51, 0.52], [0.69, 0.69, 0.69, 0.69, 0.69, 0.69]),
    lecture: 'L\'AUC oscille autour de 0,5 (pile ou face) et la perte ne bouge pas. Le CNN ne trouve aucun indice utile. Il faut chercher la cause (nombre d\'images, réglages, données) plutôt que d\'attendre.',
  },
]

/** Synthèse globale d'un entraînement terminé, en phrases simples. */
export function syntheseEntrainement(journal: EpoqueEntrainement[], rapport: RapportEntrainement, patience = 5): string[] {
  const out: string[] = []
  const lu = lireProgression(journal, patience)
  const v = rapport.validation
  const total = journal.reduce((s, e) => s + (e.duree_s || 0), 0)
  const retenue = rapport.modele?.epoque_retenue ?? lu?.epoqueMeilleure
  if (lu) {
    out.push(`Apprentissage : ${epoques(journal.length)}${total ? ` en ${duree(total)}` : ''}. L'AUC par sein en ajustement est passée de ${fr(lu.premiere)} à ${fr(lu.meilleure)} (${signe(lu.gainTotal)}) : ${Math.round(lu.premiere * 100)} puis ${Math.round(lu.meilleure * 100)} paires bien classées sur 100.`)
    if (retenue) out.push(`Version gardée : celle de l'époque ${retenue}, la meilleure en ajustement${retenue < journal.length ? ` ; les ${epoques(journal.length - retenue)} suivantes n'ont plus apporté de progrès` : ''}.`)
  }
  if (v?.auc_sein !== null && v?.auc_sein !== undefined) {
    const ecart = lu ? v.auc_sein - lu.meilleure : 0
    out.push(`Examen final, sur ${v.seins} seins de patientes jamais vues (dont ${v.seins_malins} malins) : AUC par sein ${fr(v.auc_sein)}, ${appreciation(v.auc_sein)} ; ${surCent(v.auc_sein)}.`)
    if (lu) out.push(Math.abs(ecart) <= 0.03
      ? 'Le résultat final est proche de celui de l\'ajustement : le CNN se comporte de la même façon sur des patientes nouvelles, ce qui est bon signe.'
      : ecart < 0
        ? `Le résultat final est plus bas qu'en ajustement (${signe(ecart)}) : une partie des progrès ne se retrouve pas sur des patientes nouvelles. À confirmer avec plus d'images.`
        : `Le résultat final est plus haut qu'en ajustement (${signe(ecart)}) : sans doute un effet du hasard du tirage des patientes ; à confirmer avec plus d'images.`)
  }
  if (v?.sensibilite_sein != null && v?.specificite_sein != null) {
    const se = Math.round(v.sensibilite_sein * 100), sp = Math.round(v.specificite_sein * 100)
    out.push(`Au seuil choisi : sensibilité ${fr(v.sensibilite_sein, 2)}, soit ${se} seins malins signalés sur 100 (${100 - se} manqués) ; spécificité ${fr(v.specificite_sein, 2)}, soit ${sp} seins bénins reconnus sur 100 (${100 - sp} fausses alertes).`)
  }
  if (v && v.seins < 200) out.push(`Attention : ${v.seins} seins seulement en validation, le résultat est encore imprécis (quelques points d'AUC de marge d'erreur).`)
  return out
}

/** Appréciation en mots d'une AUC (repères courants). */
export function appreciation(auc: number): string {
  if (auc < 0.6) return 'à peine mieux que le hasard'
  if (auc < 0.7) return 'faible'
  if (auc < 0.8) return 'correcte'
  if (auc < 0.9) return 'bonne'
  return 'excellente'
}
