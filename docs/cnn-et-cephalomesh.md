# CNN et CephaloMesh

Ce que fait chaque partie de MammoMesh, et pourquoi les combiner. Outil de recherche : aucun résultat ne sert à un diagnostic.

## En bref

MammoMesh réunit deux parties qui travaillent en équipe : le CNN, qui regarde la mammographie, et CephaloMesh, qui réfléchit à ce que le CNN a vu. On peut les comparer à deux radiologues : le premier a un œil très affûté mais juge chaque image isolément ; le second, plus expérimenté, recoupe les informations avant de conclure.

Cette page explique ce que fait chacun, ce qui lui manque, et pourquoi les combiner donne un meilleur résultat que chacun seul. C'est un outil de recherche : aucun résultat ne sert à un diagnostic.

## Le CNN : l'œil

Un CNN (réseau de neurones convolutif) est un modèle d'apprentissage automatique, une branche de l'intelligence artificielle, conçu pour les images. C'est un empilement de couches de calcul dont les réglages (les poids) ne sont pas écrits à la main : ils sont appris à partir d'exemples.

- Comment il regarde : il parcourt l'image par petits morceaux. Ses premiers étages repèrent des éléments simples (bords, taches claires, contrastes) ; les étages suivants les assemblent en formes plus parlantes (amas de petits points, contour irrégulier, masse).
- Comment il apprend : on lui montre des milliers de mammographies dont le diagnostic est connu (confirmé par biopsie). À chaque erreur, il ajuste un peu ses millions de réglages internes, appelés poids. Un passage complet sur toutes les images d'exemple s'appelle une époque.
- Ce qu'il produit : un score de suspicion et, à terme, les zones de l'image qui l'ont alerté, chacune avec un degré de confiance.
- Dans MammoMesh : le point de départ est EfficientNet-B0, un réseau qui sait déjà reconnaître des images courantes, réentraîné sur les mammographies de la banque CMMD. Pour l'instant, il donne un score par image ; CMMD n'indiquant pas le contour des lésions, les zones affichées montrent seulement où il a regardé (carte d'attention), pas des lésions délimitées.

Ses forces : il repère des détails visuels très fins, sans fatigue, et vite. Ses limites : il juge chaque image séparément. Il ne sait pas que deux images montrent le même sein sous deux angles, il ne compare pas le sein gauche au sein droit, ni l'examen du jour à celui d'il y a deux ans. Un défaut sur une seule image peut le tromper, et il explique mal pourquoi il s'alerte.

## CephaloMesh : le raisonnement

CephaloMesh reprend ce que le CNN a repéré et le recoupe, comme un radiologue qui ne s'arrête pas à une première impression : il vérifie.

- Une carte du sein : chaque sein est représenté comme une cible centrée sur le mamelon, découpée en cases (au départ 12 parts et 8 cercles, avec une couche par type d'indice). Chaque indice trouvé par le CNN est placé dans sa case.
- Une onde qui se propage : chaque indice envoie un signal, d'autant plus fort que le CNN était sûr de lui. Ce signal gagne les cases voisines en s'affaiblissant avec la distance. Quand plusieurs indices se renforcent au même endroit et dépassent un seuil, la case « s'allume » : la zone devient suspecte.
- Quatre vérifications de radiologue : la lésion se voit-elle sur les deux vues du même sein (CC et MLO) ? Le sein gauche et le sein droit se ressemblent-ils ? La zone a-t-elle changé depuis l'examen précédent ? La description de la lésion est-elle cohérente avec le vocabulaire BI-RADS ?
- Un second regard ciblé : quand une zone reste douteuse, CephaloMesh demande au CNN de la réexaminer en pleine résolution, puis refait son raisonnement avec ce nouveau résultat.
- Ce qu'il produit : un score par zone et par sein, une catégorie BI-RADS suggérée, et un récit qui dit pourquoi chaque zone a été retenue ou écartée.

Ses forces : il recoupe les informations, chacune de ses étapes est traçable, et il est très rapide (quelques dixièmes de milliseconde par examen). Sa limite : il ne voit pas l'image lui-même. Sans le CNN, il n'a rien sur quoi raisonner.

## Pourquoi les combiner

Chacun comble le manque de l'autre : le CNN apporte la vue, CephaloMesh apporte le jugement.

| Besoin | CNN seul | CephaloMesh seul | Les deux ensemble |
|---|---|---|---|
| Repérer un détail sur l'image | Oui | Non : il ne voit pas l'image | Oui |
| Recouper les deux vues, les deux seins et l'examen précédent | Non | Oui | Oui |
| Expliquer une alerte | Difficile | Oui, étape par étape | Oui |
| Temps de calcul | Élevé si tout est analysé en pleine résolution | Très faible | Pleine résolution seulement là où elle sert |

- Moins de fausses alertes : une tache vue sur une seule image, ou présente à l'identique dans les deux seins, ne suffit plus à déclencher une alerte.
- Plus de cancers trouvés : des indices faibles, qui ne convaincraient pas seuls, se confirment entre eux ; et le second regard en pleine résolution rattrape des détails fins comme les microcalcifications (minuscules dépôts de calcium).
- Plus rapide : la pleine résolution, coûteuse, n'est utilisée que sur les zones douteuses, pas sur toute l'image.
- Plus lisible : chaque alerte s'accompagne de ses raisons, que le radiologue peut vérifier.

Un exemple. Le CNN signale une petite tache, avec une confiance moyenne, sur la vue de dessus du sein gauche. Seul, il hésiterait. CephaloMesh constate qu'une tache semblable apparaît au même endroit sur la vue de biais, qu'il n'y a rien de comparable dans le sein droit, et qu'elle n'existait pas sur l'examen d'il y a deux ans. Ces trois indices se renforcent : la zone est retenue et réexaminée en pleine résolution. À l'inverse, une tache vue sur une seule image, présente des deux côtés et inchangée depuis des années, est affaiblie : sans doute rien d'inquiétant.

## Le parcours d'un examen

1. Passe rapide : le CNN analyse les quatre images de l'examen (deux vues par sein) en résolution réduite et signale les zones qui l'interpellent.
2. Carte : chaque zone signalée est placée dans sa case, sur la cible du sein concerné.
3. Raisonnement : l'onde se propage et les quatre vérifications renforcent ou affaiblissent chaque zone.
4. Second regard : les zones encore douteuses sont réexaminées par le CNN en pleine résolution, puis le raisonnement est refait avec ces résultats.
5. Synthèse : un score par zone et par sein, une catégorie BI-RADS suggérée et les raisons de chaque décision. C'est une suggestion pour un radiologue, jamais une décision.

## Pourquoi faire évoluer le couple

Le couple a beaucoup de réglages : la prudence de CephaloMesh, la portée de son onde, les vérifications actives, le nombre de seconds regards permis, la structure du CNN… Leurs effets se combinent, et les essayer tous à la main serait impossible.

C'est le rôle des campagnes : elles créent des variantes du couple (les individus) en changeant un réglage à la fois (une mutation), mesurent chacune, et gardent les meilleurs compromis entre cancers trouvés et rapidité. Une variante qui trouve trop peu de cancers est écartée d'office, quelle que soit sa rapidité.

Les deux parties évoluent à deux vitesses. CephaloMesh se teste en un instant : ses réglages évoluent vite. Le CNN doit être réentraîné à chaque changement de structure : il évolue lentement. Chaque code de mutation est expliqué dans le lexique des mutations.

## Où en est le projet

- Le CNN est réel : un premier modèle s'entraîne sur la carte graphique (GPU) du serveur, menu Entraînements.
- Le menu Images peut tester une image avec ce CNN entraîné (service mammomesh-inference) : score de l'image et zones où il a regardé.
- CephaloMesh et la boucle d'évolution sont encore simulés : les campagnes fonctionnent, mais leurs mesures n'ont pas de valeur tant que le CNN entraîné n'y est pas branché.
- Dans tous les cas, la décision reste humaine : l'évolution propose des candidats, seule une personne habilitée peut en retenir un.
