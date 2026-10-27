// Barre latérale de l'interface connectée : groupes et entrées, dans l'ordre du parcours
// (entraîner le CNN, chercher le raisonnement, essayer sur une image, se documenter).
// Source unique du menu (layouts/suivi.vue) ; l'accueil (utils/accueil.ts) explique chaque entrée dans le même ordre.
import type { Permissions } from '~/types'

export type Icone = 'accueil' | 'entrainement' | 'campagnes' | 'ajouter' | 'image' | 'livre' | 'cle' | 'compte' | 'sortie'

export interface EntreeNav { titre: string; to: string; icone: Icone; permission?: keyof Permissions }
export interface GroupeNav { titre?: string; entrees: EntreeNav[] }

export const navigation: GroupeNav[] = [
  { entrees: [{ titre: 'Accueil', to: '/suivi', icone: 'accueil' }] },
  { titre: 'Apprentissage du CNN', entrees: [{ titre: 'Entraînements', to: '/suivi/entrainements', icone: 'entrainement' }] },
  {
    titre: 'Recherche CephaloMesh',
    entrees: [
      { titre: 'Campagnes', to: '/suivi/campagnes', icone: 'campagnes' },
      { titre: 'Nouvelle campagne', to: '/suivi/nouvelle', icone: 'ajouter', permission: 'proposer_config' },
    ],
  },
  { titre: 'Essais', entrees: [{ titre: 'Images', to: '/suivi/images', icone: 'image' }] },
  {
    titre: 'Ressources',
    entrees: [
      { titre: 'Documentation', to: '/suivi/documentation', icone: 'livre' },
      { titre: 'Jetons MCP', to: '/suivi/jetons', icone: 'cle' },
    ],
  },
]

/** Toutes les entrées, dans l'ordre (sans le compte, en pied de barre). */
export const entrees = navigation.flatMap(g => g.entrees)
