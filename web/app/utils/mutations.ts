// Lexique des mutations (note de conception, sections « Gènes » et « Nommer les individus »).
// Sert partout où un code apparaît : fiche d'un individu, journal, page « Fonctionnement ».

export interface Mutation {
  motif: RegExp // reconnaît le code, sens compris
  famille: string // « SEU↑ / SEU↓ », pour le lexique
  partie: 'CephaloMesh' | 'CNN' | 'Croisement'
  gene: string
  role: string // à quoi sert le réglage, en mots simples
  effet: string // ce que change la mutation, dans un sens et dans l'autre
}

export const mutations: Mutation[] = [
  {
    motif: /^SEU[↑↓]$/, famille: 'SEU↑ / SEU↓', partie: 'CephaloMesh', gene: 'Seuil de déclenchement',
    role: 'Le niveau de certitude à atteindre avant que CephaloMesh juge un indice suspect. C\'est le réglage principal de sa prudence.',
    effet: '↑ relevé : CephaloMesh devient plus exigeant ; moins de fausses alertes, mais il peut rater une petite lésion. ↓ abaissé : il s\'alerte plus facilement ; il rate moins de lésions, mais se trompe plus souvent.',
  },
  {
    motif: /^TMP[↑↓][A-D]?$/, famille: 'TMP↑ / TMP↓ (suffixe A à D)', partie: 'CephaloMesh', gene: 'Réactivité selon la densité du sein',
    role: 'Plus le tissu du sein est dense (classes A à D, de peu dense à très dense), plus une lésion s\'y cache facilement. Ce réglage dose la réactivité de CephaloMesh pour chaque classe séparément : TMP↑C ne concerne que les seins de classe C.',
    effet: '↑ plus réactif, ↓ plus calme, uniquement pour la classe de densité indiquée ; les autres classes ne changent pas.',
  },
  {
    motif: /^LAM[↑↓]$/, famille: 'LAM↑ / LAM↓', partie: 'CephaloMesh', gene: 'Portée des liens',
    role: 'Un indice repéré à un endroit peut renforcer les indices voisins. Ce réglage fixe jusqu\'à quelle distance cette influence s\'étend.',
    effet: '↑ portée plus longue : des indices assez éloignés se soutiennent entre eux. ↓ portée plus courte : chaque indice est jugé surtout pour lui-même.',
  },
  {
    motif: /^RAY[↑↓]$/, famille: 'RAY↑ / RAY↓', partie: 'CephaloMesh', gene: 'Tolérance entre les deux vues',
    role: 'Chaque sein est radiographié sous deux angles (vue de dessus CC et vue de biais MLO). Une même lésion n\'y apparaît pas exactement au même endroit ; ce réglage fixe l\'écart accepté pour dire « c\'est la même ». Il ne dépasse jamais l\'erreur réellement mesurée.',
    effet: '↑ plus tolérant : retrouve plus souvent une lésion sur les deux vues, mais associe parfois deux choses sans rapport. ↓ plus strict : moins d\'associations, mais plus sûres.',
  },
  {
    motif: /^BUD[↑↓]$/, famille: 'BUD↑ / BUD↓', partie: 'CephaloMesh', gene: 'Budget d\'activité',
    role: 'Quand CephaloMesh hésite sur une zone, il peut demander au CNN de la regarder de plus près. Ce réglage fixe combien de zones il peut faire réexaminer par mammographie.',
    effet: '↑ plus de seconds regards : plus de chances de trouver une lésion, mais chaque examen prend plus de temps. ↓ plus rapide, moins de seconds regards.',
  },
  {
    motif: /^SYM[+−-]$/, famille: 'SYM+ / SYM−', partie: 'CephaloMesh', gene: 'Contrôle de symétrie',
    role: 'Compare le sein gauche et le sein droit. Ils se ressemblent d\'habitude ; une différence nette et localisée, surtout nouvelle, peut être un signe d\'alerte.',
    effet: '+ ajouté : CephaloMesh fait cette vérification en plus (un peu plus de calcul). − retiré : il ne la fait plus.',
  },
  {
    motif: /^TEM[+−-]$/, famille: 'TEM+ / TEM−', partie: 'CephaloMesh', gene: 'Comparaison avec l\'examen précédent',
    role: 'Si la patiente a déjà passé une mammographie, compare avec l\'ancienne : ce qui vient d\'apparaître ou qui grossit inquiète plus que ce qui est stable depuis des années.',
    effet: '+ ajouté : tient compte de cette évolution quand un examen précédent existe. − retiré : chaque examen est jugé seul.',
  },
  {
    motif: /^CON[+−-]$/, famille: 'CON+ / CON−', partie: 'CephaloMesh', gene: 'Accord entre les deux vues',
    role: 'Une vraie lésion se voit en général sur les deux vues du même sein. Ce contrôle donne moins de poids à un indice vu sur une seule vue, sauf dans les parties du sein qu\'une seule vue peut montrer.',
    effet: '+ ajouté : moins de fausses alertes dues à un défaut d\'une seule image. − retiré : un indice vu une seule fois compte autant qu\'un autre.',
  },
  {
    motif: /^BIR[+−-]$/, famille: 'BIR+ / BIR−', partie: 'CephaloMesh', gene: 'Cohérence de la description',
    role: 'Les radiologues décrivent une lésion avec un vocabulaire commun (BI-RADS : forme, contours, densité…). Ce contrôle vérifie, comme un relecteur, que la description produite ne se contredit pas.',
    effet: '+ ajouté : écarte les descriptions incohérentes. − retiré : plus de relecture.',
  },
  {
    motif: /^CUB\d+×\d+$/, famille: 'CUB16×8', partie: 'CephaloMesh', gene: 'Découpage du sein en cases',
    role: 'Pour situer les indices, CephaloMesh découpe le sein comme une cible centrée sur le mamelon : des parts (secteurs) et des cercles (anneaux). CUB16×8 veut dire 16 parts et 8 cercles ; on part de 12 × 8.',
    effet: 'Plus de cases : position plus précise, mais plus de calcul.',
  },
  {
    motif: /^DEG[+−-]$/, famille: 'DEG+ / DEG−', partie: 'CNN', gene: 'Part du CNN réentraînée',
    role: 'Le CNN part d\'un réseau qui sait déjà reconnaître des images courantes. Ce réglage choisit combien de ses étages sont réentraînés sur les mammographies ; les autres gardent leur savoir tel quel.',
    effet: '+ plus d\'étages réentraînés : le CNN s\'adapte mieux aux mammographies, mais l\'entraînement est plus long et il risque d\'apprendre par cœur les images vues au lieu de généraliser. − l\'inverse.',
  },
  {
    motif: /^COU-[A-Z0-9]+$/, famille: 'COU-BIFPN', partie: 'CNN', gene: 'Cou du CNN (assemblage des échelles)',
    role: 'Le CNN regarde l\'image à plusieurs échelles, du détail fin à la vue d\'ensemble. Le « cou » est la partie qui rassemble ces regards ; il existe en deux variantes, FPN et BiFPN.',
    effet: 'BiFPN : l\'information circule dans les deux sens entre les échelles ; souvent meilleur pour les petites lésions, un peu plus lent.',
  },
  {
    motif: /^BLK\+$/, famille: 'BLK+', partie: 'CNN', gene: 'Bloc ajouté',
    role: 'Ajoute une couche de traitement au CNN. Au départ, elle ne change rien au résultat ; elle ne sert qu\'une fois entraînée.',
    effet: 'Le CNN peut apprendre des choses plus fines, mais met plus de temps par image.',
  },
  {
    motif: /^TET-[A-Z]+$/, famille: 'TET-SA', partie: 'CNN', gene: 'Tête de détection',
    role: 'La « tête » est la dernière partie du CNN : celle qui trace un cadre autour de chaque lésion possible. TET-SA la remplace par une variante « sans ancres », qui trace ses cadres librement au lieu de partir de tailles fixées d\'avance.',
    effet: 'Change la façon de proposer les cadres ; on ne sait pas d\'avance si c\'est mieux, c\'est mesuré à chaque fois.',
  },
  {
    motif: /^RES\d+$/, famille: 'RES1536', partie: 'CNN', gene: 'Résolution du second regard',
    role: 'Quand CephaloMesh demande au CNN de réexaminer une zone, celle-ci est agrandie à cette taille, en pixels : RES1536 = 1536 pixels.',
    effet: 'Plus haute : on voit des détails plus fins, comme les microcalcifications (minuscules dépôts de calcium), mais le réexamen coûte plus cher.',
  },
  {
    motif: /^X\(.+×.+\)$/, famille: 'X(A × B)', partie: 'Croisement', gene: 'Croisement',
    role: 'Comme un enfant hérite de ses deux parents, l\'individu mélange les réglages de deux individus existants, nommés entre parenthèses.',
    effet: 'Peut réunir les points forts de deux lignées ; l\'enfant est ensuite testé comme tous les autres.',
  },
]

/** Explication d'un code de mutation, ou undefined pour un code inconnu. */
export function expliquer(code: string): Mutation | undefined {
  return mutations.find(m => m.motif.test(code.trim()))
}

/** Infobulle courte : gène, puis effet de la mutation. */
export function infobulle(code: string): string {
  const m = expliquer(code)
  return m ? `${m.gene} (${m.partie}) : ${m.effet}` : 'Mutation sans description'
}

/**
 * Découpe un message du journal pour repérer la liste « mutations A › B › C ».
 * Renvoie des morceaux de texte et des codes, dans l'ordre.
 */
export function decouperMessage(message: string): { texte?: string; code?: string }[] {
  const m = message.match(/^(.*mutations )(.+?)(\.?)$/)
  if (!m) return [{ texte: message }]
  const out: { texte?: string; code?: string }[] = [{ texte: m[1] }]
  m[2]!.split(' › ').forEach((c, i) => {
    if (i > 0) out.push({ texte: ' › ' })
    out.push({ code: c })
  })
  if (m[3]) out.push({ texte: m[3] })
  return out
}
