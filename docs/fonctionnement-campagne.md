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

## Une génération, pas à pas

1. Avant de commencer, l'orchestrateur vérifie les critères d'arrêt : plafond de générations, budget atteint, stagnation de E-val.
2. Le calcul avance par lots. À chaque lot, il publie la progression (lot a sur b, budget consommé) et vérifie que la campagne est toujours en cours.
3. Population : chaque individu est l'enfant d'un individu du front précédent, tiré au hasard, avec m = 1 + Poisson(μ) mutations, plafonné par « Mutations au plus ». À la première génération, les individus sont des fondateurs.
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

## Ce qui changera avec la vraie boucle

- Deux vitesses : le CNN évolue lentement (des heures par évaluation), CephaloMesh rapidement (0,2 à 0,5 ms par examen), sur des sorties du CNN mises en cache.
- Côté CNN, les enfants montent par paliers asynchrones (un pli partiel, un pli, tous les plis) ; seuls les meilleurs vont plus loin.
- La population CephaloMesh d'un nouveau CNN démarre depuis l'archive de son parent.
- Sélection NSGA-II sur les couples CNN + CephaloMesh, avec deux objectifs (sensibilité, coût) et des contraintes, dont la non-infériorité par type de lésion.
- Îles, choix adaptatif des opérateurs de mutation et bascule vers des macro-mutations quand le front ne bouge plus sur E-val pendant 8 générations.

## Lexique des mutations

Chaque individu porte la liste des mutations qui l'ont produit. Un code dit quel gène a changé et dans quel sens : ↑ relevé, ↓ abaissé, + ajouté, − retiré.

| Code | Partie | Gène | Rôle | Effet de la mutation |
|---|---|---|---|---|
| `SEU↑ / SEU↓` | CephaloMesh | Seuil de déclenchement | Signal qu'un neurone du cube doit recevoir pour s'activer ; règle la sensibilité du raisonnement. | ↑ relevé : moins d'activations, donc moins de fausses alertes, au risque de laisser passer une lésion discrète. ↓ abaissé : l'inverse. |
| `TMP↑ / TMP↓ (suffixe A à D)` | CephaloMesh | Température par densité | Une température par densité mammaire (A à D, estimée sur les images) : module la réactivité du raisonnement selon le profil du sein. TMP↑C agit sur les seins de densité C. | ↑ ou ↓ : rend le raisonnement plus ou moins réactif pour ce seul profil de densité, sans toucher aux autres. |
| `LAM↑ / LAM↓` | CephaloMesh | Portée des liens (λ) | Jusqu'où un constat influence ses voisins dans le cube (poids des liens W = exp(−d/λ)). | ↑ influence plus lointaine : des constats proches se renforcent davantage. ↓ raisonnement plus local. |
| `RAY↑ / RAY↓` | CephaloMesh | Rayon des stimuli | Tolérance à l'écart de position d'une même lésion entre les vues CC et MLO ; borné par l'erreur d'appariement mesurée. | ↑ apparie plus facilement les deux vues (plus de concordances, y compris fortuites). ↓ appariement plus strict. |
| `BUD↑ / BUD↓` | CephaloMesh | Budget d'activité | Part des zones que CephaloMesh peut renvoyer au CNN pour une réanalyse fine. | ↑ plus de réanalyses : sensibilité potentielle plus haute, coût par examen plus élevé. ↓ plus rapide, moins de seconds regards. |
| `SYM+ / SYM−` | CephaloMesh | Contrôle de symétrie | Compare les deux seins et signale une asymétrie nouvelle ou focale. | + ajouté : un raisonnement de plus (asymétries repérées, un peu plus de calcul). − retiré : l'inverse. |
| `TEM+ / TEM−` | CephaloMesh | Contrôle temporel | Compare à l'examen antérieur : apparition, croissance ou stabilité d'un constat. | + ajouté : tient compte de l'évolution dans le temps quand un antérieur existe. − retiré : l'inverse. |
| `CON+ / CON−` | CephaloMesh | Contrôle de concordance des vues | Affaiblit un constat vu sur une seule vue, sauf dans une zone visible sur une seule incidence. | + ajouté : moins de fausses alertes vues une seule fois. − retiré : l'inverse. |
| `BIR+ / BIR−` | CephaloMesh | Contrôle de cohérence BI-RADS | Vérifie que la combinaison de caractéristiques décrite est plausible. | + ajouté : écarte les combinaisons incohérentes. − retiré : l'inverse. |
| `CUB16×8` | CephaloMesh | Découpage du cube | Nombre de secteurs et d'anneaux du cube qui situe les constats autour du mamelon (12 × 8 au départ). | Découpage plus fin : localisation plus précise, plus de neurones donc plus de calcul. |
| `DEG+ / DEG−` | CNN | Étages dégelés | Nombre d'étages du CNN réentraînés plutôt que figés. | + adapte davantage le CNN aux données, au prix d'un entraînement plus long et d'un risque de surapprentissage. − l'inverse. |
| `COU-BIFPN` | CNN | Cou du CNN | Partie qui fusionne les cartes du CNN à plusieurs échelles (FPN ou BiFPN). | BiFPN : fusion bidirectionnelle, souvent meilleure sur les petites lésions, un peu plus coûteuse. |
| `BLK+` | CNN | Bloc ajouté | Ajoute un bloc à l'identique, sans changer ce que le CNN calcule au départ. | Plus de capacité après entraînement, plus de calcul par image. |
| `TET-SA` | CNN | Tête de détection | Remplace la tête par une tête sans ancres. | Change la façon de proposer les boîtes ; effet mesuré au cas par cas. |
| `RES1536` | CNN | Résolution de réanalyse | Résolution, en pixels, des zones réanalysées par le CNN à la demande de CephaloMesh. | Plus haute : détails plus fins (microcalcifications), réanalyse plus coûteuse. |
| `X(A × B)` | Croisement | Croisement | L'individu combine les génomes de deux parents, nommés entre parenthèses. | Réunit des qualités venues de deux lignées ; le résultat est mesuré comme celui de tout enfant. |
