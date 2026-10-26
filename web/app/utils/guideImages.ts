// Guide de l'écran « Images » (/suivi/images). Source de la documentation et de docs/guide-images.md.
import type { ElementGuide } from '~/utils/guideSuivi'

export interface ZoneImages { cle: string; titre: string; resume: string; elements: ElementGuide[]; aSavoir?: string }

export const zonesImages: ZoneImages[] = [
  {
    cle: 'import',
    titre: 'Importer une mammographie',
    resume: 'Charger un fichier DICOM pour y tester des individus.',
    elements: [
      { nom: 'Libellé', description: 'Nom affiché dans la liste ; à défaut, modalité, latéralité et vue (« MG L CC »). Le nom du fichier n\'est jamais repris : il peut contenir celui d\'une patiente.' },
      { nom: 'Choisir un fichier DICOM, ou le déposer', description: 'Le fichier est lu sur le serveur. Seuls les pixels et des données techniques sont gardés (dimensions, bits, vue, latéralité, fabricant, modèle, fenêtre d\'affichage) ; nom, identifiant, dates, établissement et tous les autres en-têtes sont écartés à la lecture et jamais stockés.' },
      { nom: 'Refus', description: 'Images à annotations incrustées (BurnedInAnnotation = YES), images en couleur, fichiers illisibles et compressions non prises en charge (JPEG 2000, JPEG-LS). Formats lus : non compressé et JPEG de base ; 300 Mo au plus.' },
      { nom: 'Doublons', description: 'Une image aux pixels identiques à une image déjà importée n\'est pas dupliquée : l\'existante est sélectionnée.' },
      { nom: 'Exemples synthétiques', description: 'Quatre mammographies générées (CC et MLO, gauche et droite), sans patiente réelle, à télécharger puis importer pour essayer l\'écran. Leurs en-têtes portent de fausses données patient, qui disparaissent à l\'import.' },
    ],
    aSavoir: 'Importer et tester est ouvert à tous les rôles sauf le lecteur ; chaque import et chaque test sont inscrits au journal d\'audit.',
  },
  {
    cle: 'liste',
    titre: 'Images importées',
    resume: 'Les images disponibles, la plus récente en tête.',
    elements: [
      { nom: 'Ligne d\'image', description: 'Libellé, dimensions en pixels, vue et latéralité, nombre de tests et taille stockée. Un clic l\'affiche dans la visionneuse.' },
      { nom: 'Supprimer', description: 'Efface définitivement l\'image, ses pixels et ses tests, après confirmation. Réservé à l\'administrateur et au data manager.' },
    ],
  },
  {
    cle: 'visionneuse',
    titre: 'Visionneuse',
    resume: 'L\'image en pleine résolution (16 bits), avec les outils de lecture courants.',
    elements: [
      { nom: 'Ajuster, 1:1, +, −', description: 'Image entière dans le cadre, taille réelle (un pixel d\'image par pixel d\'écran), zoom avant et arrière. La molette zoome autour du pointeur ; glisser déplace l\'image ; double-clic : ajuster.' },
      { nom: 'Fenêtrage (centre, largeur)', description: 'Choisit la plage de valeurs affichée du noir au blanc : centre = niveau moyen, largeur = contraste (plus étroite, plus contrastée). Curseurs sous l\'image, ou clic droit glissé (Maj + glisser) : horizontalement la largeur, verticalement le centre.' },
      { nom: 'Inverser, Fenêtre d\'origine', description: 'Inverse les niveaux de gris ; « Fenêtre d\'origine » revient à la fenêtre du fichier (ou, à défaut, aux centiles 1 % et 99 %) et au sens d\'origine.' },
      { nom: 'Zones', description: 'Montre ou masque les zones du test affiché : cadre rouge numéroté avec son score ; survolée, la zone passe en jaune.' },
      { nom: 'Clavier', description: 'La visionneuse sélectionnée répond à + et − (zoom), 0 (ajuster), i (inverser), f (fenêtre d\'origine), z (zones).' },
    ],
  },
  {
    cle: 'test',
    titre: 'Tester un individu',
    resume: 'Choisir une version du couple CNN + CephaloMesh et la faire travailler sur l\'image affichée.',
    elements: [
      { nom: 'Campagne', description: 'Par défaut, la campagne terminée ou arrêtée la plus récente qui a des individus réalisables.' },
      { nom: 'Génération', description: 'Par défaut la génération finale ; toute génération intermédiaire avec un front peut être choisie. Les générations sans individu réalisable ne sont pas sélectionnables.' },
      { nom: 'Individu du front', description: 'Les individus du front de la génération, avec leur sensibilité sur E-val et leur coût ; par défaut le meilleur sur E-val. Ses mutations s\'expliquent au survol.' },
      { nom: 'Tester sur cette image (simulation)', description: 'Lance le test et affiche le résultat : score de l\'image, temps de calcul, zones signalées (type et score), superposées à l\'image.' },
      { nom: 'Bandeau « Simulation »', description: 'Tant que le CNN et CephaloMesh ne sont pas branchés, le résultat est tiré de façon déterministe à partir de l\'image et de l\'individu : il n\'a aucune valeur, ni diagnostique ni de recherche.' },
      { nom: 'Tests précédents', description: 'Historique des tests sur cette image (individu, génération, score, date) ; un clic réaffiche un résultat.' },
    ],
    aSavoir: 'Le même individu sur la même image donne toujours le même résultat simulé ; un individu plus sensible signale en moyenne plus de zones.',
  },
]
