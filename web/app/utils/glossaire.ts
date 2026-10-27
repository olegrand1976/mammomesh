// Glossaire : sigles et mots complexes de l'interface. Source unique des infobulles (composants Sigle et
// TexteGlossaire : chaque sigle d'un écran affiche son explication au survol ou au focus), de la page
// Documentation › Glossaire et de docs/glossaire.md (un test vérifie la concordance).

export interface Terme {
  terme: string
  /** Forme développée d'un sigle (souvent en anglais). */
  forme?: string
  definition: string
  /** Sigle : infobulle automatique partout où il apparaît ; mot : glossaire seulement. */
  sigle: boolean
}

export const glossaire: Terme[] = [
  // ---------- Sigles ----------
  { terme: 'AUC', forme: 'Area Under the Curve, aire sous la courbe ROC', sigle: true,
    definition: 'Qualité d\'un score, de 0,5 (hasard) à 1 (parfait) : la probabilité qu\'un cas malin reçoive un score plus haut qu\'un cas bénin.' },
  { terme: 'BI-RADS', forme: 'Breast Imaging Reporting and Data System', sigle: true,
    definition: 'Classification de l\'American College of Radiology, de 0 à 6, qui exprime le degré de suspicion d\'une image mammaire.' },
  { terme: 'CBIS-DDSM', forme: 'Curated Breast Imaging Subset of the Digital Database for Screening Mammography', sigle: true,
    definition: 'Jeu public de films de mammographie numérisés, avec contours des lésions et pathologie (TCIA, licence CC BY 3.0).' },
  { terme: 'CC', forme: 'cranio-caudale', sigle: true,
    definition: 'Incidence de mammographie prise de haut en bas. Avec la MLO, l\'une des deux vues standard de chaque sein.' },
  { terme: 'CC BY', forme: 'Creative Commons Attribution', sigle: true,
    definition: 'Licence libre : usage, y compris commercial, et redistribution permis, à condition de citer la source.' },
  { terme: 'CE', forme: 'Conformité européenne', sigle: true,
    definition: 'Marquage exigé pour mettre un dispositif médical sur le marché européen (règlement (UE) 2017/745) ; MammoMesh, outil de recherche, ne l\'a pas.' },
  { terme: 'CLAIM', forme: 'Checklist for Artificial Intelligence in Medical Imaging', sigle: true,
    definition: 'Liste de contrôle des publications sur l\'IA en imagerie médicale ; les comptes rendus de MammoMesh la suivent.' },
  { terme: 'TRIPOD', forme: 'Transparent Reporting of a multivariable prediction model for Individual Prognosis Or Diagnosis', sigle: true,
    definition: 'Règles de compte rendu des modèles de prédiction ; TRIPOD+AI les étend aux modèles d\'apprentissage automatique.' },
  { terme: 'CMMD', forme: 'Chinese Mammography Database', sigle: true,
    definition: 'Jeu public de mammographies numériques de 1 775 patientes, bénin ou malin prouvé par biopsie (TCIA, licence CC BY 4.0). Sert au premier CNN.' },
  { terme: 'CNN', forme: 'Convolutional Neural Network, réseau de neurones convolutif', sigle: true,
    definition: 'Modèle d\'apprentissage automatique (une branche de l\'intelligence artificielle) conçu pour les images : des couches de calcul dont les réglages, les poids, sont appris sur des exemples. Dans MammoMesh, c\'est le « regard » : il lit la mammographie et donne un score.' },
  { terme: 'CPU', forme: 'Central Processing Unit', sigle: true,
    definition: 'Processeur principal de la machine. Sans GPU, l\'entraînement s\'y fait, mais très lentement.' },
  { terme: 'CSAW-CC', forme: 'Cohort of Screen-Aged Women, Case-Control', sigle: true,
    definition: 'Jeu suédois (Karolinska) avec l\'historique de dépistage complet de 8 723 femmes ; accès sur demande, licence CC BY 4.0.' },
  { terme: 'CUDA', forme: 'Compute Unified Device Architecture', sigle: true,
    definition: 'Plateforme de calcul des GPU NVIDIA, utilisée par PyTorch pour entraîner le CNN sur la carte graphique.' },
  { terme: 'DICOM', forme: 'Digital Imaging and Communications in Medicine', sigle: true,
    definition: 'Format standard des images médicales : pixels et en-têtes (patiente, appareil, vue…). MammoMesh n\'en garde que les pixels et des données techniques.' },
  { terme: 'E-ajust', forme: 'ensemble d\'ajustement', sigle: true,
    definition: 'Patientes mises de côté pour suivre les progrès, choisir les meilleurs individus et le seuil. Jamais utilisées pour apprendre.' },
  { terme: 'E-val', forme: 'ensemble de validation', sigle: true,
    definition: 'Patientes jamais vues pendant l\'entraînement ni la sélection : la mesure honnête, faite à la fin.' },
  { terme: 'EMBED', forme: 'Emory Breast Imaging Dataset', sigle: true,
    definition: 'Grand jeu américain (480 000 images en accès ouvert) ; licence non commerciale, écarté pour l\'instant.' },
  { terme: 'FRDR', forme: 'Federated Research Data Repository', sigle: true,
    definition: 'Dépôt canadien de données de recherche qui héberge NLBS ; téléchargement par Globus.' },
  { terme: 'Grad-CAM', forme: 'Gradient-weighted Class Activation Mapping', sigle: true,
    definition: 'Carte d\'attention d\'un CNN : elle montre les régions de l\'image qui ont le plus pesé dans son score. Ce n\'est pas une détection de lésion.' },
  { terme: 'GPU', forme: 'Graphics Processing Unit, carte graphique', sigle: true,
    definition: 'Processeur massivement parallèle : il entraîne le CNN des dizaines de fois plus vite qu\'un CPU.' },
  { terme: 'JPEG', forme: 'Joint Photographic Experts Group', sigle: true,
    definition: 'Famille de compressions d\'image ; certaines variantes (JPEG 2000, JPEG-LS) ne sont pas encore lues à l\'import.' },
  { terme: 'MCP', forme: 'Model Context Protocol', sigle: true,
    definition: 'Protocole qui permet à Claude Code d\'interroger MammoMesh (résultats, brouillons) avec un jeton personnel.' },
  { terme: 'MLO', forme: 'médio-latérale oblique', sigle: true,
    definition: 'Incidence de mammographie prise en biais, qui montre aussi le creux de l\'aisselle. Avec la CC, l\'une des deux vues standard.' },
  { terme: 'NBIA', forme: 'National Biomedical Imaging Archive', sigle: true,
    definition: 'Interface de téléchargement de TCIA, utilisée par le script de la banque.' },
  { terme: 'NLBS', forme: 'Newfoundland and Labrador Breast Screening', sigle: true,
    definition: 'Jeu canadien de dépistage réel : 26 988 images, 149 cancers, sans contours de lésions (FRDR, licence CC BY 4.0).' },
  { terme: 'NSGA-II', forme: 'Non-dominated Sorting Genetic Algorithm II', sigle: true,
    definition: 'Algorithme génétique qui classe une population par fronts de Pareto pour optimiser plusieurs objectifs à la fois.' },
  { terme: 'OMI-DB', forme: 'OPTIMAM Mammography Image Database', sigle: true,
    definition: 'Base britannique de plus de 10 millions d\'images ; licence à négocier, écartée pour l\'instant.' },
  { terme: 'PCG', forme: 'Permuted Congruential Generator', sigle: true,
    definition: 'Générateur de nombres pseudo-aléatoires : avec la même graine, il redonne exactement le même tirage.' },
  { terme: 'ROCm', forme: 'Radeon Open Compute', sigle: true,
    definition: 'Plateforme de calcul des GPU AMD, l\'équivalent de CUDA.' },
  { terme: 'ROI', forme: 'Region Of Interest, région d\'intérêt', sigle: true,
    definition: 'Zone d\'une image qui contient une lésion ; un masque ROI la délimite au pixel.' },
  { terme: 'RSNA-SMBC', forme: 'RSNA Screening Mammography Breast Cancer Detection', sigle: true,
    definition: 'Jeu du concours RSNA 2022 (environ 78 000 images) ; licence non commerciale, écarté pour l\'instant.' },
  { terme: 'SSE', forme: 'Server-Sent Events', sigle: true,
    definition: 'Flux du serveur vers le navigateur : c\'est ce qui met l\'écran de suivi d\'une campagne à jour en direct.' },
  { terme: 'TCIA', forme: 'The Cancer Imaging Archive', sigle: true,
    definition: 'Archive publique américaine d\'imagerie du cancer : source de CMMD et de CBIS-DDSM.' },
  { terme: 'WSL', forme: 'Windows Subsystem for Linux', sigle: true,
    definition: 'Linux intégré à Windows 11 : c\'est par lui que la pile MammoMesh tourne sur un poste Windows.' },

  { terme: 'AMD-V', forme: 'AMD Virtualization', sigle: true,
    definition: 'Virtualisation matérielle des processeurs AMD, à activer dans le BIOS/UEFI pour WSL et Docker.' },
  { terme: 'API', forme: 'Application Programming Interface', sigle: true,
    definition: 'Le service (écrit en Go) que l\'interface et Claude Code interrogent ; il vérifie les droits et lit la base.' },
  { terme: 'ASHA', forme: 'Asynchronous Successive Halving Algorithm', sigle: true,
    definition: 'Évaluation par paliers : chaque candidat est d\'abord essayé à petit coût, seuls les meilleurs montent au palier suivant.' },
  { terme: 'AWS', forme: 'Amazon Web Services', sigle: true,
    definition: 'Nuage d\'Amazon ; certains jeux d\'images (RSNA, EMBED) y sont distribués.' },
  { terme: 'BIFPN', forme: 'Bidirectional Feature Pyramid Network', sigle: true,
    definition: 'Variante de FPN où les informations circulent dans les deux sens entre les échelles.' },
  { terme: 'BIOS', forme: 'Basic Input/Output System', sigle: true,
    definition: 'Micrologiciel de démarrage de la machine (aujourd\'hui l\'UEFI) : c\'est là qu\'on active la virtualisation.' },
  { terme: 'CSV', forme: 'Comma-Separated Values', sigle: true,
    definition: 'Fichier de tableau en texte, une ligne par enregistrement ; le journal des époques en est un.' },
  { terme: 'FNV-1a', forme: 'Fowler–Noll–Vo, variante 1a', sigle: true,
    definition: 'Fonction de hachage rapide ; ici, elle initialise le tirage simulé pour qu\'une même image donne toujours le même résultat.' },
  { terme: 'FPN', forme: 'Feature Pyramid Network', sigle: true,
    definition: '« Cou » d\'un CNN qui combine des cartes de caractéristiques de plusieurs résolutions, utile pour les petites lésions.' },
  { terme: 'HTTPS', forme: 'HTTP Secure', sigle: true,
    definition: 'Web chiffré ; le site MammoMesh n\'est servi qu\'en HTTPS, avec le certificat de son autorité interne (Caddy).' },
  { terme: 'JPEG-LS', forme: 'JPEG Lossless', sigle: true,
    definition: 'Compression sans perte pour les images médicales ; pas encore lue à l\'import.' },
  { terme: 'MG', forme: 'mammographie (modalité DICOM)', sigle: true,
    definition: 'Code de modalité DICOM des mammographies ; les autres modalités (CT, IRM…) sont acceptées mais signalées.' },
  { terme: 'NAS', forme: 'Network Attached Storage', sigle: true,
    definition: 'Stockage en réseau ; il peut accueillir la banque d\'images et les copies de sauvegarde.' },
  { terme: 'NSGA-III', forme: 'Non-dominated Sorting Genetic Algorithm III', sigle: true,
    definition: 'Successeur de NSGA-II pour trois objectifs ou plus : la diversité est guidée par des points de référence.' },
  { terme: 'PDF', forme: 'Portable Document Format', sigle: true,
    definition: 'Document imprimable ; la procédure d\'installation sur Windows 11 existe aussi en PDF.' },
  { terme: 'RAM', forme: 'Random Access Memory, mémoire vive', sigle: true,
    definition: 'Mémoire de travail de la machine ; elle borne le nombre de workers qui préparent les images.' },
  { terme: 'ROC', forme: 'Receiver Operating Characteristic', sigle: true,
    definition: 'Courbe sensibilité contre taux de faux positifs quand on fait varier le seuil ; l\'AUC en est l\'aire.' },
  { terme: 'RSNA', forme: 'Radiological Society of North America', sigle: true,
    definition: 'Société savante de radiologie ; elle a organisé en 2022 un concours de détection du cancer du sein.' },
  { terme: 'SHA-256', forme: 'Secure Hash Algorithm, 256 bits', sigle: true,
    definition: 'Empreinte d\'un contenu : deux configurations identiques ont la même ; la moindre différence la change entièrement.' },
  { terme: 'SSH', forme: 'Secure Shell', sigle: true,
    definition: 'Connexion chiffrée à une machine ; une clé SSH permet aussi de récupérer le dépôt GitHub sans mot de passe.' },
  { terme: 'TCP', forme: 'Transmission Control Protocol', sigle: true,
    definition: 'Protocole réseau des connexions web ; le site écoute sur les ports TCP 80 et 443.' },
  { terme: 'UEFI', forme: 'Unified Extensible Firmware Interface', sigle: true,
    definition: 'Micrologiciel de démarrage moderne (successeur du BIOS), où l\'on active la virtualisation.' },
  { terme: 'VT-x', forme: 'Intel Virtualization Technology', sigle: true,
    definition: 'Virtualisation matérielle des processeurs Intel, à activer dans le BIOS/UEFI pour WSL et Docker.' },
  { terme: 'XLSX', forme: 'classeur Excel', sigle: true,
    definition: 'Format de tableur ; les données cliniques de CMMD sont fournies ainsi.' },

  // ---------- Mots ----------
  { terme: 'Accumulation de gradient', sigle: false,
    definition: 'Additionner les corrections de plusieurs petits lots avant de les appliquer : on obtient l\'effet d\'un grand lot sans la mémoire qu\'il demanderait.' },
  { terme: 'Arrêt anticipé', sigle: false,
    definition: 'Fin d\'un entraînement avant le nombre d\'époques prévu, quand l\'AUC en ajustement ne progresse plus pendant « patience » époques.' },
  { terme: 'Budget (cœur-heures)', sigle: false,
    definition: 'Calcul qu\'une campagne a le droit de consommer : une heure d\'un cœur de processeur vaut une cœur-heure.' },
  { terme: 'Calibrage', sigle: false,
    definition: 'Mesure, sur le GPU lui-même, du plus grand lot d\'images qui tient en mémoire ; le résultat est gardé pour les fois suivantes.' },
  { terme: 'CephaloMesh', sigle: false,
    definition: 'Le « raisonnement » de MammoMesh : il reprend ce que le CNN a vu et le contrôle (concordance des vues, symétrie, évolution, cohérence BI-RADS).' },
  { terme: 'Descente de gradient', sigle: false,
    definition: 'Méthode d\'apprentissage : après chaque lot d\'images, chaque poids du CNN est corrigé un peu dans le sens qui réduit l\'erreur.' },
  { terme: 'EfficientNet-B0', sigle: false,
    definition: 'Architecture de CNN légère et précise, utilisée pour le premier modèle ; elle part de poids appris sur ImageNet.' },
  { terme: 'Époque', sigle: false,
    definition: 'Un passage complet sur toutes les images d\'entraînement.' },
  { terme: 'Fenêtrage', sigle: false,
    definition: 'Choix de la plage de niveaux affichée du noir au blanc (centre et largeur) pour lire une image médicale.' },
  { terme: 'Front de Pareto', sigle: false,
    definition: 'Ensemble des compromis qu\'aucun autre ne bat sur tous les objectifs à la fois : ici, plus sensible ou plus rapide, sans être pire sur l\'autre.' },
  { terme: 'Génération', sigle: false,
    definition: 'Une étape d\'une campagne : une nouvelle population d\'individus est produite, mesurée, puis triée.' },
  { terme: 'Graine', sigle: false,
    definition: 'Nombre de départ du hasard : avec la même graine, le même tirage se répète exactement.' },
  { terme: 'ImageNet', sigle: false,
    definition: 'Grande base de photos ordinaires. Des poids appris dessus donnent au CNN un bon point de départ (licence non commerciale).' },
  { terme: 'Inférence', sigle: false,
    definition: 'Utilisation d\'un modèle déjà entraîné pour donner un score à une nouvelle image, sans rien apprendre. Dans MammoMesh, le service mammomesh-inference fait ce calcul pour le menu Images.' },
  { terme: 'Individu', sigle: false,
    definition: 'Une version complète du couple CNN + CephaloMesh, décrite par son génome (ses réglages), mesurée puis comparée aux autres.' },
  { terme: 'Latéralité', sigle: false,
    definition: 'Côté du sein sur l\'image : gauche (L) ou droit (R).' },
  { terme: 'Lot', sigle: false,
    definition: 'Groupe d\'images traitées ensemble par le GPU (batch) ; sa taille dépend de la mémoire disponible.' },
  { terme: 'Mutation', sigle: false,
    definition: 'Petite modification d\'un réglage d\'un individu pour produire un descendant ; chaque code de mutation est expliqué dans le lexique.' },
  { terme: 'Perte', sigle: false,
    definition: 'Erreur moyenne du CNN sur les images d\'apprentissage : elle pénalise une mauvaise réponse d\'autant plus qu\'il en est sûr. Elle doit baisser d\'époque en époque ; seule, elle ne dit pas si le CNN réussit sur des images nouvelles (c\'est le rôle de l\'AUC d\'ajustement).' },
  { terme: 'Plancher de sensibilité', sigle: false,
    definition: 'Sensibilité minimale exigée : un individu en dessous n\'est pas réalisable, quel que soit son temps de calcul.' },
  { terme: 'Point de reprise', sigle: false,
    definition: 'Sauvegarde écrite à la fin de chaque époque : un entraînement coupé repart de là au lieu de recommencer.' },
  { terme: 'Pseudonymisation', sigle: false,
    definition: 'Retrait des données qui identifient une patiente (nom, identifiant, dates, établissement) ; seuls les pixels et des données techniques restent.' },
  { terme: 'Réalisable', sigle: false,
    definition: 'Se dit d\'un individu dont la sensibilité (borne basse) atteint le plancher diminué de la marge.' },
  { terme: 'Sensibilité', sigle: false,
    definition: 'Part des cancers effectivement repérés. Une sensibilité de 0,85 : 85 cancers trouvés sur 100.' },
  { terme: 'Seuil', sigle: false,
    definition: 'Score au-dessus duquel un sein (ou, dans le menu Images, une image) est déclaré suspect ; il est choisi en ajustement pour atteindre la sensibilité cible.' },
  { terme: 'Spécificité', sigle: false,
    definition: 'Part des cas non malins correctement écartés. Plus elle est haute, moins il y a de fausses alertes.' },
  { terme: 'Stagnation', sigle: false,
    definition: 'Arrêt d\'une campagne quand la meilleure sensibilité en validation ne progresse plus pendant le nombre de générations fixé.' },
  { terme: 'Taux d\'apprentissage', sigle: false,
    definition: 'Taille des corrections à chaque pas. Trop grand : apprentissage instable ; trop petit : très lent.' },
  { terme: 'Taux de rappel', sigle: false,
    definition: 'Part des femmes dépistées rappelées pour examens complémentaires ; la sensibilité est mesurée à un taux de rappel cible.' },
  { terme: 'Veilleur', sigle: false,
    definition: 'Service du serveur qui exécute les demandes de la page Entraînements et reprend un entraînement coupé dès que le GPU est libre.' },
]

const parTerme = new Map(glossaire.map(t => [t.terme, t]))
export const terme = (nom: string): Terme | undefined => parTerme.get(nom)

/** Ancre d'un terme dans la page Glossaire. */
export const ancre = (nom: string) => 'g-' + nom.normalize('NFD').replace(/[̀-ͯ]/g, '').toLowerCase().replace(/[^a-z0-9]+/g, '-').replace(/^-|-$/g, '')

// Les sigles les plus longs d'abord (CC BY avant CC), entiers seulement (pas « CC » dans « CCD »).
const sigles = glossaire.filter(t => t.sigle).map(t => t.terme).sort((a, b) => b.length - a.length)
const echapper = (s: string) => s.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')
const MOTIF = new RegExp(`(?<![\\p{L}\\p{N}_-])(${sigles.map(echapper).join('|')})(?![\\p{L}\\p{N}_-])`, 'gu')

/** Découpe un texte en morceaux simples et en sigles du glossaire. */
export function decouper(texte: string): { texte: string; sigle?: Terme }[] {
  const out: { texte: string; sigle?: Terme }[] = []
  let dernier = 0
  for (const m of texte.matchAll(MOTIF)) {
    if (m.index! > dernier) out.push({ texte: texte.slice(dernier, m.index) })
    out.push({ texte: m[0], sigle: parTerme.get(m[0]) })
    dernier = m.index! + m[0].length
  }
  if (dernier < texte.length) out.push({ texte: texte.slice(dernier) })
  return out
}
