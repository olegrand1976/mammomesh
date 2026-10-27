# Glossaire

Dans l'application : menu « Documentation » › « Glossaire » (`/suivi/documentation/glossaire`). Ce document en reprend le contenu. Dans chaque écran, un sigle souligné en pointillé affiche cette explication au survol ou au clavier.

## Sigles

**AMD-V** (AMD Virtualization) : Virtualisation matérielle des processeurs AMD, à activer dans le BIOS/UEFI pour WSL et Docker.

**API** (Application Programming Interface) : Le service (écrit en Go) que l'interface et Claude Code interrogent ; il vérifie les droits et lit la base.

**ASHA** (Asynchronous Successive Halving Algorithm) : Évaluation par paliers : chaque candidat est d'abord essayé à petit coût, seuls les meilleurs montent au palier suivant.

**AUC** (Area Under the Curve, aire sous la courbe ROC) : Qualité d'un score, de 0,5 (hasard) à 1 (parfait) : la probabilité qu'un cas malin reçoive un score plus haut qu'un cas bénin.

**AWS** (Amazon Web Services) : Nuage d'Amazon ; certains jeux d'images (RSNA, EMBED) y sont distribués.

**BI-RADS** (Breast Imaging Reporting and Data System) : Classification de l'American College of Radiology, de 0 à 6, qui exprime le degré de suspicion d'une image mammaire.

**BIFPN** (Bidirectional Feature Pyramid Network) : Variante de FPN où les informations circulent dans les deux sens entre les échelles.

**BIOS** (Basic Input/Output System) : Micrologiciel de démarrage de la machine (aujourd'hui l'UEFI) : c'est là qu'on active la virtualisation.

**CBIS-DDSM** (Curated Breast Imaging Subset of the Digital Database for Screening Mammography) : Jeu public de films de mammographie numérisés, avec contours des lésions et pathologie (TCIA, licence CC BY 3.0).

**CC** (cranio-caudale) : Incidence de mammographie prise de haut en bas. Avec la MLO, l'une des deux vues standard de chaque sein.

**CC BY** (Creative Commons Attribution) : Licence libre : usage, y compris commercial, et redistribution permis, à condition de citer la source.

**CE** (Conformité européenne) : Marquage exigé pour mettre un dispositif médical sur le marché européen (règlement (UE) 2017/745) ; MammoMesh, outil de recherche, ne l'a pas.

**CLAIM** (Checklist for Artificial Intelligence in Medical Imaging) : Liste de contrôle des publications sur l'IA en imagerie médicale ; les comptes rendus de MammoMesh la suivent.

**CMMD** (Chinese Mammography Database) : Jeu public de mammographies numériques de 1 775 patientes, bénin ou malin prouvé par biopsie (TCIA, licence CC BY 4.0). Sert au premier CNN.

**CNN** (Convolutional Neural Network, réseau de neurones convolutif) : Modèle d'apprentissage automatique (une branche de l'intelligence artificielle) conçu pour les images : des couches de calcul dont les réglages, les poids, sont appris sur des exemples. Dans MammoMesh, c'est le « regard » : il lit la mammographie et donne un score.

**CPU** (Central Processing Unit) : Processeur principal de la machine. Sans GPU, l'entraînement s'y fait, mais très lentement.

**CSAW-CC** (Cohort of Screen-Aged Women, Case-Control) : Jeu suédois (Karolinska) avec l'historique de dépistage complet de 8 723 femmes ; accès sur demande, licence CC BY 4.0.

**CSV** (Comma-Separated Values) : Fichier de tableau en texte, une ligne par enregistrement ; le journal des époques en est un.

**CUDA** (Compute Unified Device Architecture) : Plateforme de calcul des GPU NVIDIA, utilisée par PyTorch pour entraîner le CNN sur la carte graphique.

**DICOM** (Digital Imaging and Communications in Medicine) : Format standard des images médicales : pixels et en-têtes (patiente, appareil, vue…). MammoMesh n'en garde que les pixels et des données techniques.

**E-ajust** (ensemble d'ajustement) : Patientes mises de côté pour suivre les progrès, choisir les meilleurs individus et le seuil. Jamais utilisées pour apprendre.

**E-val** (ensemble de validation) : Patientes jamais vues pendant l'entraînement ni la sélection : la mesure honnête, faite à la fin.

**EMBED** (Emory Breast Imaging Dataset) : Grand jeu américain (480 000 images en accès ouvert) ; licence non commerciale, écarté pour l'instant.

**FNV-1a** (Fowler–Noll–Vo, variante 1a) : Fonction de hachage rapide ; ici, elle initialise le tirage simulé pour qu'une même image donne toujours le même résultat.

**FPN** (Feature Pyramid Network) : « Cou » d'un CNN qui combine des cartes de caractéristiques de plusieurs résolutions, utile pour les petites lésions.

**FRDR** (Federated Research Data Repository) : Dépôt canadien de données de recherche qui héberge NLBS ; téléchargement par Globus.

**GPU** (Graphics Processing Unit, carte graphique) : Processeur massivement parallèle : il entraîne le CNN des dizaines de fois plus vite qu'un CPU.

**Grad-CAM** (Gradient-weighted Class Activation Mapping) : Carte d'attention d'un CNN : elle montre les régions de l'image qui ont le plus pesé dans son score. Ce n'est pas une détection de lésion.

**HTTPS** (HTTP Secure) : Web chiffré ; le site MammoMesh n'est servi qu'en HTTPS, avec le certificat de son autorité interne (Caddy).

**JPEG** (Joint Photographic Experts Group) : Famille de compressions d'image ; certaines variantes (JPEG 2000, JPEG-LS) ne sont pas encore lues à l'import.

**JPEG-LS** (JPEG Lossless) : Compression sans perte pour les images médicales ; pas encore lue à l'import.

**MCP** (Model Context Protocol) : Protocole qui permet à Claude Code d'interroger MammoMesh (résultats, brouillons) avec un jeton personnel.

**MG** (mammographie (modalité DICOM)) : Code de modalité DICOM des mammographies ; les autres modalités (CT, IRM…) sont acceptées mais signalées.

**MLO** (médio-latérale oblique) : Incidence de mammographie prise en biais, qui montre aussi le creux de l'aisselle. Avec la CC, l'une des deux vues standard.

**NAS** (Network Attached Storage) : Stockage en réseau ; il peut accueillir la banque d'images et les copies de sauvegarde.

**NBIA** (National Biomedical Imaging Archive) : Interface de téléchargement de TCIA, utilisée par le script de la banque.

**NLBS** (Newfoundland and Labrador Breast Screening) : Jeu canadien de dépistage réel : 26 988 images, 149 cancers, sans contours de lésions (FRDR, licence CC BY 4.0).

**NSGA-II** (Non-dominated Sorting Genetic Algorithm II) : Algorithme génétique qui classe une population par fronts de Pareto pour optimiser plusieurs objectifs à la fois.

**NSGA-III** (Non-dominated Sorting Genetic Algorithm III) : Successeur de NSGA-II pour trois objectifs ou plus : la diversité est guidée par des points de référence.

**OMI-DB** (OPTIMAM Mammography Image Database) : Base britannique de plus de 10 millions d'images ; licence à négocier, écartée pour l'instant.

**PCG** (Permuted Congruential Generator) : Générateur de nombres pseudo-aléatoires : avec la même graine, il redonne exactement le même tirage.

**PDF** (Portable Document Format) : Document imprimable ; la procédure d'installation sur Windows 11 existe aussi en PDF.

**RAM** (Random Access Memory, mémoire vive) : Mémoire de travail de la machine ; elle borne le nombre de workers qui préparent les images.

**ROC** (Receiver Operating Characteristic) : Courbe sensibilité contre taux de faux positifs quand on fait varier le seuil ; l'AUC en est l'aire.

**ROCm** (Radeon Open Compute) : Plateforme de calcul des GPU AMD, l'équivalent de CUDA.

**ROI** (Region Of Interest, région d'intérêt) : Zone d'une image qui contient une lésion ; un masque ROI la délimite au pixel.

**RSNA** (Radiological Society of North America) : Société savante de radiologie ; elle a organisé en 2022 un concours de détection du cancer du sein.

**RSNA-SMBC** (RSNA Screening Mammography Breast Cancer Detection) : Jeu du concours RSNA 2022 (environ 78 000 images) ; licence non commerciale, écarté pour l'instant.

**SHA-256** (Secure Hash Algorithm, 256 bits) : Empreinte d'un contenu : deux configurations identiques ont la même ; la moindre différence la change entièrement.

**SSE** (Server-Sent Events) : Flux du serveur vers le navigateur : c'est ce qui met l'écran de suivi d'une campagne à jour en direct.

**SSH** (Secure Shell) : Connexion chiffrée à une machine ; une clé SSH permet aussi de récupérer le dépôt GitHub sans mot de passe.

**TCIA** (The Cancer Imaging Archive) : Archive publique américaine d'imagerie du cancer : source de CMMD et de CBIS-DDSM.

**TCP** (Transmission Control Protocol) : Protocole réseau des connexions web ; le site écoute sur les ports TCP 80 et 443.

**TRIPOD** (Transparent Reporting of a multivariable prediction model for Individual Prognosis Or Diagnosis) : Règles de compte rendu des modèles de prédiction ; TRIPOD+AI les étend aux modèles d'apprentissage automatique.

**UEFI** (Unified Extensible Firmware Interface) : Micrologiciel de démarrage moderne (successeur du BIOS), où l'on active la virtualisation.

**VT-x** (Intel Virtualization Technology) : Virtualisation matérielle des processeurs Intel, à activer dans le BIOS/UEFI pour WSL et Docker.

**WSL** (Windows Subsystem for Linux) : Linux intégré à Windows 11 : c'est par lui que la pile MammoMesh tourne sur un poste Windows.

**XLSX** (classeur Excel) : Format de tableur ; les données cliniques de CMMD sont fournies ainsi.

## Mots

**Accumulation de gradient** : Additionner les corrections de plusieurs petits lots avant de les appliquer : on obtient l'effet d'un grand lot sans la mémoire qu'il demanderait.

**Arrêt anticipé** : Fin d'un entraînement avant le nombre d'époques prévu, quand l'AUC en ajustement ne progresse plus pendant « patience » époques.

**Budget (cœur-heures)** : Calcul qu'une campagne a le droit de consommer : une heure d'un cœur de processeur vaut une cœur-heure.

**Calibrage** : Mesure, sur le GPU lui-même, du plus grand lot d'images qui tient en mémoire ; le résultat est gardé pour les fois suivantes.

**CephaloMesh** : Le « raisonnement » de MammoMesh : il reprend ce que le CNN a vu et le contrôle (concordance des vues, symétrie, évolution, cohérence BI-RADS).

**Descente de gradient** : Méthode d'apprentissage : après chaque lot d'images, chaque poids du CNN est corrigé un peu dans le sens qui réduit l'erreur.

**EfficientNet-B0** : Architecture de CNN légère et précise, utilisée pour le premier modèle ; elle part de poids appris sur ImageNet.

**Époque** : Un passage complet sur toutes les images d'entraînement.

**Fenêtrage** : Choix de la plage de niveaux affichée du noir au blanc (centre et largeur) pour lire une image médicale.

**Front de Pareto** : Ensemble des compromis qu'aucun autre ne bat sur tous les objectifs à la fois : ici, plus sensible ou plus rapide, sans être pire sur l'autre.

**Génération** : Une étape d'une campagne : une nouvelle population d'individus est produite, mesurée, puis triée.

**Graine** : Nombre de départ du hasard : avec la même graine, le même tirage se répète exactement.

**ImageNet** : Grande base de photos ordinaires. Des poids appris dessus donnent au CNN un bon point de départ (licence non commerciale).

**Individu** : Une version complète du couple CNN + CephaloMesh, décrite par son génome (ses réglages), mesurée puis comparée aux autres.

**Inférence** : Utilisation d'un modèle déjà entraîné pour donner un score à une nouvelle image, sans rien apprendre. Dans MammoMesh, le service mammomesh-inference fait ce calcul pour le menu Images.

**Latéralité** : Côté du sein sur l'image : gauche (L) ou droit (R).

**Lot** : Groupe d'images traitées ensemble par le GPU (batch) ; sa taille dépend de la mémoire disponible.

**Mutation** : Petite modification d'un réglage d'un individu pour produire un descendant ; chaque code de mutation est expliqué dans le lexique.

**Perte** : Erreur moyenne du CNN sur les images d'apprentissage : elle pénalise une mauvaise réponse d'autant plus qu'il en est sûr. Elle doit baisser d'époque en époque ; seule, elle ne dit pas si le CNN réussit sur des images nouvelles (c'est le rôle de l'AUC d'ajustement).

**Plancher de sensibilité** : Sensibilité minimale exigée : un individu en dessous n'est pas réalisable, quel que soit son temps de calcul.

**Point de reprise** : Sauvegarde écrite à la fin de chaque époque : un entraînement coupé repart de là au lieu de recommencer.

**Pseudonymisation** : Retrait des données qui identifient une patiente (nom, identifiant, dates, établissement) ; seuls les pixels et des données techniques restent.

**Réalisable** : Se dit d'un individu dont la sensibilité (borne basse) atteint le plancher diminué de la marge.

**Sensibilité** : Part des cancers effectivement repérés. Une sensibilité de 0,85 : 85 cancers trouvés sur 100.

**Seuil** : Score au-dessus duquel un sein (ou, dans le menu Images, une image) est déclaré suspect ; il est choisi en ajustement pour atteindre la sensibilité cible.

**Spécificité** : Part des cas non malins correctement écartés. Plus elle est haute, moins il y a de fausses alertes.

**Stagnation** : Arrêt d'une campagne quand la meilleure sensibilité en validation ne progresse plus pendant le nombre de générations fixé.

**Taux d'apprentissage** : Taille des corrections à chaque pas. Trop grand : apprentissage instable ; trop petit : très lent.

**Taux de rappel** : Part des femmes dépistées rappelées pour examens complémentaires ; la sensibilité est mesurée à un taux de rappel cible.

**Veilleur** : Service du serveur qui exécute les demandes de la page Entraînements et reprend un entraînement coupé dès que le GPU est libre.
