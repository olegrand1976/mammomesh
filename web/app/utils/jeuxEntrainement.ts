// Jeux d'entraînement (banque du jalon 0) : source unique de la page Documentation › Jeux d'entraînement
// (/suivi/documentation/jeux-entrainement) et de docs/jeux-entrainement.md (un test vérifie la concordance,
// et que chaque collection de examples/manifeste.json y figure).
import type { Partie } from '~/utils/fonctionnement'

export const etatJeux = 'État au 27 octobre 2026'

export const partiesJeux: Partie[] = [
  {
    id: 'en-bref',
    titre: 'En bref',
    blocs: [
      { p: 'La banque d\'entraînement réunit des mammographies publiques pour entraîner et évaluer le CNN, puis CephaloMesh. C\'est le jalon 0 : tant qu\'elle n\'est pas constituée, la boucle CNN + CephaloMesh reste simulée. C\'est un outil de recherche : aucun résultat ne sert à un diagnostic.' },
      { p: 'Critère de choix : une licence qui permet aussi un usage commercial (CC BY), car un usage commercial n\'est pas exclu à l\'avenir. Les jeux sous licence non commerciale sont écartés pour l\'instant, même s\'ils sont plus grands.' },
    ],
  },
  {
    id: 'retenus',
    titre: 'Jeux retenus',
    blocs: [
      {
        table: {
          entetes: ['Jeu', 'Contenu', 'Volume', 'Licence', 'Accès'],
          lignes: [
            ['CBIS-DDSM', 'États-Unis, films numérisés : 1 566 patientes, 10 239 images (complètes, découpes, masques ROI des lésions), pathologie, descriptions CSV.', '~164 Go', 'CC BY 3.0', 'Libre, TCIA (API NBIA)'],
            ['CMMD', 'Chine, mammographie numérique : 1 775 patientes, bénin ou malin prouvé par biopsie, sous-types moléculaires pour 749 patientes, données cliniques XLSX.', '~23 Go', 'CC BY 4.0', 'Libre, TCIA (API NBIA)'],
            ['NLBS', 'Canada (Terre-Neuve-et-Labrador), dépistage réel, version 3 : 5 997 examens, 26 988 images DICOM 16 bits, 149 cancers prouvés par biopsie, 1 516 faux positifs, 4 332 normaux ; sans annotation de lésion.', '~350 Go (archives)', 'CC BY 4.0', 'Libre, FRDR (Globus)'],
          ],
        },
      },
    ],
  },
  {
    id: 'usages',
    titre: 'À quoi sert chaque jeu',
    blocs: [
      {
        points: [
          'CBIS-DDSM : localiser les lésions, grâce aux masques ROI (masses et microcalcifications). Ce sont des films numérisés, au rendu différent du numérique : à réserver à l\'apprentissage de la localisation, pas à la mesure finale.',
          'CMMD : distinguer bénin et malin sur des images numériques, avec une preuve par biopsie.',
          'NLBS : la répartition réelle d\'un programme de dépistage (peu de cancers, beaucoup de normaux et de faux positifs) ; sert à mesurer la sensibilité au taux de rappel cible et à régler le seuil sans annotation de lésion.',
        ],
      },
    ],
  },
  {
    id: 'telechargement',
    titre: 'Téléchargement',
    blocs: [
      { p: 'Le script examples/telecharger.sh récupère les collections décrites dans la clé « banque » de examples/manifeste.json : ./examples/telecharger.sh --banque CMMD (ou CBIS-DDSM, NLBS, tout), avec BANQUE_DIR pour choisir le disque de destination.' },
      {
        points: [
          'Espace disque : avant chaque collection, le script calcule ce qui reste à télécharger et refuse de démarrer si le disque ne peut pas tout recevoir en gardant MARGE_GO libres (10 Go par défaut). Il recontrôle avant chaque série et s\'arrête proprement si la place manque. Pour NLBS, il contrôle aussi la taille décompressée avant l\'extraction.',
          'Reprise : relancer la même commande. Les séries terminées (marqueur .ok, nombre d\'images vérifié) sont sautées ; les échecs sont listés dans tcia/COLLECTION/.echecs.',
          'NLBS : FRDR ne publie pas de lien direct. Le script vérifie la place, indique le lien Globus, puis extrait les archives .7z déposées dans BANQUE_DIR/nlbs/archives.',
          'Arborescence : tcia/COLLECTION/PATIENTE/SÉRIE/*.dcm (identifiants TCIA), la liste TCIA (series.json) et les métadonnées (metadonnees/) ; nlbs/donnees/ pour NLBS.',
        ],
      },
    ],
  },
  {
    id: 'a-demander',
    titre: 'À demander',
    blocs: [
      {
        table: {
          entetes: ['Jeu', 'Contenu', 'Volume', 'Licence', 'Démarche'],
          lignes: [
            ['CSAW-CC', 'Suède (Karolinska) : 8 723 femmes dont 873 cancers, tout l\'historique de dépistage jusqu\'au diagnostic, lésions segmentées au pixel, signes visibles sur l\'examen antérieur.', '~2,5 To', 'CC BY 4.0', 'Demande à kidr@ki.se. Prioritaire : les examens antérieurs servent au contrôle temporel de CephaloMesh.'],
            ['VinDr-Mammo', 'Vietnam : 5 000 examens à 4 vues (20 000 images), BI-RADS, densité, boîtes des lésions.', 'non publié', 'PhysioNet Restricted Health Data 1.5.0', 'Compte PhysioNet et accord d\'utilisation ; clause sur l\'usage commercial à vérifier avant la demande.'],
          ],
        },
      },
    ],
  },
  {
    id: 'ecartes',
    titre: 'Écartés pour l\'instant',
    blocs: [
      {
        table: {
          entetes: ['Jeu', 'Contenu', 'Raison'],
          lignes: [
            ['RSNA-SMBC', 'Emory et BreastScreen Victoria : 19 418 examens à 4 vues (~78 000 images), 784 cancers.', 'Licence non commerciale, sans redistribution.'],
            ['EMBED (Open Data)', 'Emory : 22 382 patientes, 76 373 examens, 480 323 images, lésions décrites et résultats d\'analyse, examens antérieurs.', 'Licence non commerciale ; publier un modèle ou des poids entraînés demande un accord écrit.'],
            ['OPTIMAM (OMI-DB)', 'Royaume-Uni : plus de 10 millions d\'images.', 'Licence à négocier avec Cancer Research Horizons, probablement payante.'],
          ],
        },
      },
    ],
  },
  {
    id: 'regles',
    titre: 'Règles d\'usage',
    blocs: [
      {
        points: [
          'Citer chaque jeu utilisé (licences CC BY) dans toute publication ou tout rapport.',
          'Aucune tentative de réidentification des patientes ; les données restent sur le réseau local, sans redistribution.',
          'Séparer entraînement, ajustement (E-ajust) et validation (E-val) par patiente, jamais par image, pour qu\'une même patiente ne se retrouve pas des deux côtés.',
          'Garder hors entraînement les ensembles de test officiels des jeux qui en publient (CBIS-DDSM : Mass-Test et Calc-Test).',
          'Contrôler l\'espace disque avant tout téléchargement.',
        ],
      },
    ],
  },
  {
    id: 'citations',
    titre: 'Citations',
    blocs: [
      {
        points: [
          'Sawyer-Lee R., Gimenez F., Hoogi A., Rubin D. Curated Breast Imaging Subset of DDSM (CBIS-DDSM). The Cancer Imaging Archive, 2016. doi:10.7937/K9/TCIA.2016.7O02S9CY',
          'Cui C., Li L., Cai H. et al. The Chinese Mammography Database (CMMD). The Cancer Imaging Archive, 2021. doi:10.7937/tcia.eqde-4b16',
          'Full Field Digital Mammography Dataset from a Population Screening Program. Scientific Data, 2025. doi:10.1038/s41597-025-05866-0 ; données NL-Breast-Screen, FRDR, doi:10.20383/103.01526',
          'Clark K. et al. The Cancer Imaging Archive (TCIA): Maintaining and Operating a Public Information Repository. J Digit Imaging 2013;26(6):1045-1057.',
          'Dembrower K., Lindholm P., Strand F. A Multi-million Mammography Image Dataset and Population-Based Screening Cohort for the Training and Evaluation of Deep Neural Networks—the Cohort of Screen-Aged Women (CSAW). J Digit Imaging 2020;33:408-413. (si CSAW-CC est obtenu)',
        ],
      },
    ],
  },
]

export interface SourceJeu { titre: string; url: string }

// Pages consultées pour établir ce document.
export const sourcesJeux: SourceJeu[] = [
  { titre: 'CBIS-DDSM (TCIA)', url: 'https://www.cancerimagingarchive.net/collection/cbis-ddsm/' },
  { titre: 'CMMD (TCIA)', url: 'https://www.cancerimagingarchive.net/collection/cmmd/' },
  { titre: 'NL-Breast-Screen, version 3 (FRDR)', url: 'https://www.frdr-dfdr.ca/repo/dataset/2af839d4-d54f-48d7-9a1a-a00e57fe72ea' },
  { titre: 'Article NL-Breast-Screen (Scientific Data, 2025)', url: 'https://www.nature.com/articles/s41597-025-05866-0' },
  { titre: 'CSAW-CC (Swedish National Data Service)', url: 'https://researchdata.se/en/catalogue/dataset/2021-204-1' },
  { titre: 'VinDr-Mammo (PhysioNet)', url: 'https://www.physionet.org/content/vindr-mammo/' },
  { titre: 'RSNA-SMBC (Registry of Open Data on AWS)', url: 'https://registry.opendata.aws/rsna-screening-mammography-breast-cancer-detection/' },
  { titre: 'Article RSNA-SMBC (Radiology: AI, 2026)', url: 'https://pubs.rsna.org/doi/10.1148/ryai.250375' },
  { titre: 'EMBED Open Data (GitHub)', url: 'https://github.com/Emory-HITI/EMBED_Open_Data' },
  { titre: 'Licence EMBED', url: 'https://github.com/Emory-HITI/EMBED_Open_Data/blob/main/EMBED_license.md' },
  { titre: 'OPTIMAM / OMI-DB (Cancer Research Horizons)', url: 'https://www.cancerresearchhorizons.com/licensing-opportunities/optimam-mammography-image-database-omi-db' },
]
