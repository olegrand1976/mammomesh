# Fonctionnement d'une campagne

Dans l'application : menu « Fonctionnement » (`/suivi/fonctionnement`). Ce document en reprend le contenu.

## En bref

Une campagne fait évoluer une population de configurations CephaloMesh, le raisonnement qui suit le CNN, pour trouver les meilleurs compromis entre sensibilité et temps de calcul, sous un plancher de sensibilité. Elle tourne seule, génération après génération, jusqu'à un critère d'arrêt fixé avant le lancement. C'est un outil de recherche : aucun résultat ne sert à un diagnostic.

Dans cette version du socle, la boucle est simulée. L'orchestrateur, les règles de réalisabilité, de sélection et d'arrêt, le temps réel, le journal et l'audit sont réels ; les mesures (sensibilités, coûts, génomes) viennent d'une simulation déterministe, en attendant le branchement du CNN et de CephaloMesh.

## Qui fait quoi

| Composant | Rôle |
|---|---|
| Interface | Prépare, vérifie et lance les campagnes, les suit en direct et les pilote. Elle peut être fermée : rien ne s'arrête. |
| API | Vérifie les droits et les contrôles, enregistre les brouillons, relaie le temps réel aux navigateurs. |
| Orchestrateur | Seul à calculer. Une seule instance active à la fois (verrou en base) ; il regarde chaque seconde s'il a une campagne à faire avancer. |
| PostgreSQL | Garde configurations, générations, individus, journal et audit. Chaque génération y est écrite en une seule transaction. |
| Redis | Compte les tentatives de connexion ; s'il manque, des compteurs en mémoire prennent le relais. |
| Claude Code (MCP) | Lit les résultats, propose des brouillons ; lancer ou régler une campagne demande toujours une confirmation humaine. |

## Cycle de vie

1. Brouillon : la configuration est figée et reçoit une empreinte (SHA-256). La même configuration donne toujours la même empreinte.
2. Vérification : chaque réglage est comparé à sa plage ; la durée et le calcul sont estimés. Rien n'est lancé.
3. Lancement : par le responsable scientifique ou l'administrateur, une seule fois par brouillon. La campagne passe « En cours ».
4. En cours ⇄ En pause : la pause prend effet à la fin du lot en cours ; à la reprise, la génération interrompue est refaite à l'identique.
5. Fin : « Terminée » quand un critère d'arrêt est atteint, « Arrêtée » sur demande d'une personne, « En échec » si l'orchestrateur rencontre une erreur (détaillée au journal).
6. Suite : depuis une campagne terminée ou arrêtée, « Repartir de cette campagne » prépare une nouvelle campagne qui reprend ses réglages, modifiables, et dont la génération 1 descend du front de la génération choisie (la finale par défaut). Ce départ fait partie de la configuration figée et de son empreinte.

## Une génération, pas à pas

1. Avant de commencer, l'orchestrateur vérifie les critères d'arrêt : plafond de générations, budget atteint, stagnation de E-val.
2. Le calcul avance par lots. À chaque lot, il publie la progression (lot a sur b, budget consommé) et vérifie que la campagne est toujours en cours.
3. Population : chaque individu est l'enfant d'un individu du front précédent, tiré au hasard, avec m = 1 + Poisson(μ) mutations, plafonné par « Mutations au plus ». À la première génération, les individus sont des fondateurs, ou les enfants du front de départ quand la campagne repart d'une autre.
4. Mesures de chaque individu : sensibilité sur E-ajust avec son intervalle (bornes basse et haute), sensibilité sur E-val, coût par examen en millisecondes.
5. Réalisabilité : un individu est retenu si la borne basse de sa sensibilité atteint « plancher − marge ». Les autres sont écartés, et le journal le dit.
6. Front de Pareto : parmi les réalisables, ceux qu'aucun autre ne bat à la fois en sensibilité et en coût. Ils deviennent les parents de la génération suivante.
7. Meilleur : le réalisable de meilleure sensibilité sur E-val. S'il dépasse le record de la campagne d'au moins 0,1 point, le journal annonce un nouveau meilleur et le compteur de stagnation revient à 0 ; sinon le compteur augmente, avec une alerte 3 générations avant l'arrêt.
8. Écriture : génération, individus, compteurs, budget et événements du journal sont écrits en une seule transaction ; aucun écran ne voit une génération à moitié écrite.
9. Diffusion : chaque événement est notifié par PostgreSQL et relayé aux navigateurs ouverts, qui se mettent à jour seuls.

## Deux mesures : E-ajust et E-val

Les patientes du jeu d'évolution sont coupées une fois pour toutes en deux parties : E-ajust (75 %) et E-val (25 %). L'évolution n'optimise que sur E-ajust : la réalisabilité et le front de Pareto s'y calculent.

E-val ne sert jamais à optimiser. Il désigne le meilleur individu, arrête la campagne quand il ne progresse plus et choisira le front exporté. Un individu sélectionné parmi des milliers paraît toujours un peu meilleur sur les données qui ont servi à le choisir : l'écart E-ajust − E-val mesure cet optimisme. Avec environ 20 000 configurations testées, la note de conception l'estime à 2 à 3 points de sensibilité.

## Critères d'arrêt

| Critère | Règle | Réglage |
|---|---|---|
| Plafond | Le nombre de générations atteint le plafond. | Générations au plus |
| Budget | Les cœur-heures consommées atteignent le budget ; la génération commencée va jusqu'au bout, la dernière peut donc le dépasser un peu. | Budget (cœur-heures), augmentable en cours de route |
| Stagnation | N générations de suite sans nouveau meilleur sur E-val (gain d'au moins 0,1 point). Le compteur tourne aussi tant qu'aucun individu n'est réalisable. | Arrêt si E-val stable |
| Arrêt humain | « Arrêter » dans le suivi, ou l'outil MCP campagne_reglage après confirmation. | — |

Le premier critère atteint l'emporte ; le journal indique lequel.

## Reproductibilité, reprise et traçabilité

- Le tirage dépend de la graine, du numéro de campagne et du numéro de génération : une génération recalculée donne exactement le même résultat.
- Après un redémarrage du serveur, l'orchestrateur reprend chaque campagne en cours à la génération suivante et l'annonce au journal ; rien de ce qui est écrit n'est recalculé.
- La configuration d'une campagne ne change jamais après le lancement ; son empreinte l'identifie. Seul le budget peut être augmenté.
- Chaque geste (lancement, pause, reprise, budget, arrêt, appel MCP) est inscrit au journal d'audit, avec son auteur et sa source.

## Temps réel

- Les événements du journal arrivent tous, dans l'ordre, sans fusion.
- La progression des lots est fusionnée : au plus un message toutes les 2 secondes par campagne, pour ne pas saturer les navigateurs.
- Après une coupure, le navigateur se reconnecte et reprend après le dernier événement reçu ; si trop d'événements ont été manqués, l'écran relit toutes ses données de lui-même.

## Tester une version sur une image

1. Importer une mammographie DICOM (écran « Images ») : le serveur ne garde que les pixels et des données techniques ; aucun en-tête identifiant, ni le nom du fichier.
2. Choisir une campagne, une génération (la finale par défaut) et un individu de son front (le meilleur sur E-val par défaut).
3. Lancer le test : score de l'image, temps de calcul et zones signalées s'affichent sur l'image ; le test est gardé et inscrit à l'audit.

Dans ce socle, le résultat est simulé de façon déterministe à partir de l'image et de l'individu, sans aucune valeur ; le parcours est prêt pour le vrai moteur.

## Ce qui changera avec la vraie boucle

- Deux vitesses : le CNN évolue lentement (des heures par évaluation), CephaloMesh rapidement (0,2 à 0,5 ms par examen), sur des sorties du CNN mises en cache.
- Côté CNN, les enfants montent par paliers asynchrones (un pli partiel, un pli, tous les plis) ; seuls les meilleurs vont plus loin.
- La population CephaloMesh d'un nouveau CNN démarre depuis l'archive de son parent.
- Sélection NSGA-II sur les couples CNN + CephaloMesh, avec deux objectifs (sensibilité, coût) et des contraintes, dont la non-infériorité par type de lésion.
- Îles, choix adaptatif des opérateurs de mutation et bascule vers des macro-mutations quand le front ne bouge plus sur E-val pendant 8 générations.

## Lexique des mutations

MammoMesh essaie de nombreuses versions de lui-même, un peu comme une sélection naturelle accélérée. Chaque version s'appelle un individu, et ses réglages sont ses gènes. Pour créer un nouvel individu, on copie un individu existant et on change un réglage : c'est une mutation. Un code dit quel réglage a changé et dans quel sens : ↑ augmenté, ↓ diminué, + vérification ajoutée, − vérification retirée.

| Code | Partie | Réglage (gène) | À quoi il sert | Ce que change la mutation |
|---|---|---|---|---|
| `SEU↑ / SEU↓` | CephaloMesh | Seuil de déclenchement | Le niveau de certitude à atteindre avant que CephaloMesh juge un indice suspect. C'est le réglage principal de sa prudence. | ↑ relevé : CephaloMesh devient plus exigeant ; moins de fausses alertes, mais il peut rater une petite lésion. ↓ abaissé : il s'alerte plus facilement ; il rate moins de lésions, mais se trompe plus souvent. |
| `TMP↑ / TMP↓ (suffixe A à D)` | CephaloMesh | Réactivité selon la densité du sein | Plus le tissu du sein est dense (classes A à D, de peu dense à très dense), plus une lésion s'y cache facilement. Ce réglage dose la réactivité de CephaloMesh pour chaque classe séparément : TMP↑C ne concerne que les seins de classe C. | ↑ plus réactif, ↓ plus calme, uniquement pour la classe de densité indiquée ; les autres classes ne changent pas. |
| `LAM↑ / LAM↓` | CephaloMesh | Portée des liens | Un indice repéré à un endroit peut renforcer les indices voisins. Ce réglage fixe jusqu'à quelle distance cette influence s'étend. | ↑ portée plus longue : des indices assez éloignés se soutiennent entre eux. ↓ portée plus courte : chaque indice est jugé surtout pour lui-même. |
| `RAY↑ / RAY↓` | CephaloMesh | Tolérance entre les deux vues | Chaque sein est radiographié sous deux angles (vue de dessus CC et vue de biais MLO). Une même lésion n'y apparaît pas exactement au même endroit ; ce réglage fixe l'écart accepté pour dire « c'est la même ». Il ne dépasse jamais l'erreur réellement mesurée. | ↑ plus tolérant : retrouve plus souvent une lésion sur les deux vues, mais associe parfois deux choses sans rapport. ↓ plus strict : moins d'associations, mais plus sûres. |
| `BUD↑ / BUD↓` | CephaloMesh | Budget d'activité | Quand CephaloMesh hésite sur une zone, il peut demander au CNN de la regarder de plus près. Ce réglage fixe combien de zones il peut faire réexaminer par mammographie. | ↑ plus de seconds regards : plus de chances de trouver une lésion, mais chaque examen prend plus de temps. ↓ plus rapide, moins de seconds regards. |
| `SYM+ / SYM−` | CephaloMesh | Contrôle de symétrie | Compare le sein gauche et le sein droit. Ils se ressemblent d'habitude ; une différence nette et localisée, surtout nouvelle, peut être un signe d'alerte. | + ajouté : CephaloMesh fait cette vérification en plus (un peu plus de calcul). − retiré : il ne la fait plus. |
| `TEM+ / TEM−` | CephaloMesh | Comparaison avec l'examen précédent | Si la patiente a déjà passé une mammographie, compare avec l'ancienne : ce qui vient d'apparaître ou qui grossit inquiète plus que ce qui est stable depuis des années. | + ajouté : tient compte de cette évolution quand un examen précédent existe. − retiré : chaque examen est jugé seul. |
| `CON+ / CON−` | CephaloMesh | Accord entre les deux vues | Une vraie lésion se voit en général sur les deux vues du même sein. Ce contrôle donne moins de poids à un indice vu sur une seule vue, sauf dans les parties du sein qu'une seule vue peut montrer. | + ajouté : moins de fausses alertes dues à un défaut d'une seule image. − retiré : un indice vu une seule fois compte autant qu'un autre. |
| `BIR+ / BIR−` | CephaloMesh | Cohérence de la description | Les radiologues décrivent une lésion avec un vocabulaire commun (BI-RADS : forme, contours, densité…). Ce contrôle vérifie, comme un relecteur, que la description produite ne se contredit pas. | + ajouté : écarte les descriptions incohérentes. − retiré : plus de relecture. |
| `CUB16×8` | CephaloMesh | Découpage du sein en cases | Pour situer les indices, CephaloMesh découpe le sein comme une cible centrée sur le mamelon : des parts (secteurs) et des cercles (anneaux). CUB16×8 veut dire 16 parts et 8 cercles ; on part de 12 × 8. | Plus de cases : position plus précise, mais plus de calcul. |
| `DEG+ / DEG−` | CNN | Part du CNN réentraînée | Le CNN part d'un réseau qui sait déjà reconnaître des images courantes. Ce réglage choisit combien de ses étages sont réentraînés sur les mammographies ; les autres gardent leur savoir tel quel. | + plus d'étages réentraînés : le CNN s'adapte mieux aux mammographies, mais l'entraînement est plus long et il risque d'apprendre par cœur les images vues au lieu de généraliser. − l'inverse. |
| `COU-BIFPN` | CNN | Cou du CNN (assemblage des échelles) | Le CNN regarde l'image à plusieurs échelles, du détail fin à la vue d'ensemble. Le « cou » est la partie qui rassemble ces regards ; il existe en deux variantes, FPN et BiFPN. | BiFPN : l'information circule dans les deux sens entre les échelles ; souvent meilleur pour les petites lésions, un peu plus lent. |
| `BLK+` | CNN | Bloc ajouté | Ajoute une couche de traitement au CNN. Au départ, elle ne change rien au résultat ; elle ne sert qu'une fois entraînée. | Le CNN peut apprendre des choses plus fines, mais met plus de temps par image. |
| `TET-SA` | CNN | Tête de détection | La « tête » est la dernière partie du CNN : celle qui trace un cadre autour de chaque lésion possible. TET-SA la remplace par une variante « sans ancres », qui trace ses cadres librement au lieu de partir de tailles fixées d'avance. | Change la façon de proposer les cadres ; on ne sait pas d'avance si c'est mieux, c'est mesuré à chaque fois. |
| `RES1536` | CNN | Résolution du second regard | Quand CephaloMesh demande au CNN de réexaminer une zone, celle-ci est agrandie à cette taille, en pixels : RES1536 = 1536 pixels. | Plus haute : on voit des détails plus fins, comme les microcalcifications (minuscules dépôts de calcium), mais le réexamen coûte plus cher. |
| `X(A × B)` | Croisement | Croisement | Comme un enfant hérite de ses deux parents, l'individu mélange les réglages de deux individus existants, nommés entre parenthèses. | Peut réunir les points forts de deux lignées ; l'enfant est ensuite testé comme tous les autres. |
