// Lexique des mutations (note de conception, sections « Gènes » et « Nommer les individus »).
// Sert partout où un code apparaît : fiche d'un individu, journal, page « Fonctionnement ».

export interface Mutation {
  motif: RegExp // reconnaît le code, sens compris
  famille: string // « SEU↑ / SEU↓ », pour le lexique
  partie: 'CephaloMesh' | 'CNN' | 'Croisement'
  gene: string
  role: string
  effet: string // ce que change la mutation, dans un sens et dans l'autre
}

export const mutations: Mutation[] = [
  {
    motif: /^SEU[↑↓]$/, famille: 'SEU↑ / SEU↓', partie: 'CephaloMesh', gene: 'Seuil de déclenchement',
    role: 'Signal qu\'un neurone du cube doit recevoir pour s\'activer ; règle la sensibilité du raisonnement.',
    effet: '↑ relevé : moins d\'activations, donc moins de fausses alertes, au risque de laisser passer une lésion discrète. ↓ abaissé : l\'inverse.',
  },
  {
    motif: /^TMP[↑↓][A-D]?$/, famille: 'TMP↑ / TMP↓ (suffixe A à D)', partie: 'CephaloMesh', gene: 'Température par densité',
    role: 'Une température par densité mammaire (A à D, estimée sur les images) : module la réactivité du raisonnement selon le profil du sein. TMP↑C agit sur les seins de densité C.',
    effet: '↑ ou ↓ : rend le raisonnement plus ou moins réactif pour ce seul profil de densité, sans toucher aux autres.',
  },
  {
    motif: /^LAM[↑↓]$/, famille: 'LAM↑ / LAM↓', partie: 'CephaloMesh', gene: 'Portée des liens (λ)',
    role: 'Jusqu\'où un constat influence ses voisins dans le cube (poids des liens W = exp(−d/λ)).',
    effet: '↑ influence plus lointaine : des constats proches se renforcent davantage. ↓ raisonnement plus local.',
  },
  {
    motif: /^RAY[↑↓]$/, famille: 'RAY↑ / RAY↓', partie: 'CephaloMesh', gene: 'Rayon des stimuli',
    role: 'Tolérance à l\'écart de position d\'une même lésion entre les vues CC et MLO ; borné par l\'erreur d\'appariement mesurée.',
    effet: '↑ apparie plus facilement les deux vues (plus de concordances, y compris fortuites). ↓ appariement plus strict.',
  },
  {
    motif: /^BUD[↑↓]$/, famille: 'BUD↑ / BUD↓', partie: 'CephaloMesh', gene: 'Budget d\'activité',
    role: 'Part des zones que CephaloMesh peut renvoyer au CNN pour une réanalyse fine.',
    effet: '↑ plus de réanalyses : sensibilité potentielle plus haute, coût par examen plus élevé. ↓ plus rapide, moins de seconds regards.',
  },
  {
    motif: /^SYM[+−-]$/, famille: 'SYM+ / SYM−', partie: 'CephaloMesh', gene: 'Contrôle de symétrie',
    role: 'Compare les deux seins et signale une asymétrie nouvelle ou focale.',
    effet: '+ ajouté : un raisonnement de plus (asymétries repérées, un peu plus de calcul). − retiré : l\'inverse.',
  },
  {
    motif: /^TEM[+−-]$/, famille: 'TEM+ / TEM−', partie: 'CephaloMesh', gene: 'Contrôle temporel',
    role: 'Compare à l\'examen antérieur : apparition, croissance ou stabilité d\'un constat.',
    effet: '+ ajouté : tient compte de l\'évolution dans le temps quand un antérieur existe. − retiré : l\'inverse.',
  },
  {
    motif: /^CON[+−-]$/, famille: 'CON+ / CON−', partie: 'CephaloMesh', gene: 'Contrôle de concordance des vues',
    role: 'Affaiblit un constat vu sur une seule vue, sauf dans une zone visible sur une seule incidence.',
    effet: '+ ajouté : moins de fausses alertes vues une seule fois. − retiré : l\'inverse.',
  },
  {
    motif: /^BIR[+−-]$/, famille: 'BIR+ / BIR−', partie: 'CephaloMesh', gene: 'Contrôle de cohérence BI-RADS',
    role: 'Vérifie que la combinaison de caractéristiques décrite est plausible.',
    effet: '+ ajouté : écarte les combinaisons incohérentes. − retiré : l\'inverse.',
  },
  {
    motif: /^CUB\d+×\d+$/, famille: 'CUB16×8', partie: 'CephaloMesh', gene: 'Découpage du cube',
    role: 'Nombre de secteurs et d\'anneaux du cube qui situe les constats autour du mamelon (12 × 8 au départ).',
    effet: 'Découpage plus fin : localisation plus précise, plus de neurones donc plus de calcul.',
  },
  {
    motif: /^DEG[+−-]$/, famille: 'DEG+ / DEG−', partie: 'CNN', gene: 'Étages dégelés',
    role: 'Nombre d\'étages du CNN réentraînés plutôt que figés.',
    effet: '+ adapte davantage le CNN aux données, au prix d\'un entraînement plus long et d\'un risque de surapprentissage. − l\'inverse.',
  },
  {
    motif: /^COU-[A-Z0-9]+$/, famille: 'COU-BIFPN', partie: 'CNN', gene: 'Cou du CNN',
    role: 'Partie qui fusionne les cartes du CNN à plusieurs échelles (FPN ou BiFPN).',
    effet: 'BiFPN : fusion bidirectionnelle, souvent meilleure sur les petites lésions, un peu plus coûteuse.',
  },
  {
    motif: /^BLK\+$/, famille: 'BLK+', partie: 'CNN', gene: 'Bloc ajouté',
    role: 'Ajoute un bloc à l\'identique, sans changer ce que le CNN calcule au départ.',
    effet: 'Plus de capacité après entraînement, plus de calcul par image.',
  },
  {
    motif: /^TET-[A-Z]+$/, famille: 'TET-SA', partie: 'CNN', gene: 'Tête de détection',
    role: 'Remplace la tête par une tête sans ancres.',
    effet: 'Change la façon de proposer les boîtes ; effet mesuré au cas par cas.',
  },
  {
    motif: /^RES\d+$/, famille: 'RES1536', partie: 'CNN', gene: 'Résolution de réanalyse',
    role: 'Résolution, en pixels, des zones réanalysées par le CNN à la demande de CephaloMesh.',
    effet: 'Plus haute : détails plus fins (microcalcifications), réanalyse plus coûteuse.',
  },
  {
    motif: /^X\(.+×.+\)$/, famille: 'X(A × B)', partie: 'Croisement', gene: 'Croisement',
    role: 'L\'individu combine les génomes de deux parents, nommés entre parenthèses.',
    effet: 'Réunit des qualités venues de deux lignées ; le résultat est mesuré comme celui de tout enfant.',
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
