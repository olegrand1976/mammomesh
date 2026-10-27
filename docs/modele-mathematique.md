# Modèle mathématique

Dans l'application : Documentation › Modèle mathématique. Les formules exactes de ce que calcule le socle (code de l'orchestrateur, de l'API et de l'interface), puis le modèle prévu par la note de conception pour la vraie boucle.

## Notations

| Symbole | Sens | Réglage |
|---|---|---|
| g | numéro de génération (1, 2, …) | — |
| G | plafond de générations | Générations au plus |
| P | taille de la population | Population |
| π | plancher de sensibilité | Plancher de sensibilité |
| δ | marge de non-infériorité, en points | Marge (points) |
| r | taux de rappel cible | Taux de rappel cible |
| μ, m_max | paramètre et plafond du nombre de mutations | μ mutations, Mutations au plus |
| N | générations sans progrès avant arrêt | Arrêt si E-val stable |
| B | budget, en cœur-heures | Budget (cœur-heures) |
| g₀ | génération de départ quand la campagne repart d'une autre (0 sinon) | Point de départ |
| sᵢ, eᵢ | sensibilité de l'individu i sur E-ajust et sur E-val | — |
| κᵢ | coût par examen de l'individu i, en ms | — |

## Coût, budget et durée

Chaque individu est évalué sur 20 000 examens à 0,35 ms l'un (repère de la note : 0,2 à 0,5 ms). Le coût d'une génération, en cœur-heures :

$$
C = \frac{P \times 20\,000 \times 0{,}35\ \text{ms}}{3\,600\,000\ \text{ms/h}} = \frac{7P}{3600} \qquad (P = 100 \Rightarrow C \approx 0{,}194)
$$

Budget consommé après g générations, et dernière génération jouée quand le budget limite (une génération commencée va jusqu'au bout) :

$$
U_g = g\,C, \qquad g_B = \left\lceil \frac{B}{C} \right\rceil
$$

Durée simulée d'une génération et estimations « au plus » affichées avant le lancement :

$$
d = t_{\text{lot}} \times n_{\text{lots}}, \qquad D_{\max} = G\,d, \qquad C_{\max} = G\,C
$$

## Nombre de mutations

Chaque enfant reçoit m mutations, au moins une, plafonné par m_max :

$$
m = \min\bigl(1 + K,\ m_{\max}\bigr), \qquad K \sim \mathrm{Poisson}(\mu)
$$

Moyenne, affichée dans « Effet de vos réglages » (μ = 1 et m_max = 5 donnent 1,996, « un peu moins de 2 ») :

$$
\mathbb{E}[m] = \sum_{k=0}^{m_{\max}-2} (1+k)\, e^{-\mu}\frac{\mu^k}{k!} \;+\; m_{\max}\Bigl(1 - \sum_{k=0}^{m_{\max}-2} e^{-\mu}\frac{\mu^k}{k!}\Bigr)
$$

## Simulation d'un individu (socle actuel)

En attendant le CNN et CephaloMesh, chaque individu est tiré ainsi. Le plafond de sensibilité monte et ralentit ; une campagne qui repart d'une autre prolonge son avancée :

$$
c(g) = 0{,}80 + 0{,}10\,\bigl(1 - e^{-(g + g_0)/35}\bigr)
$$

Coût, sensibilité sur E-ajust (plus coûteux ⇒ un peu plus sensible), intervalle et sensibilité sur E-val :

$$
\kappa_i \sim \mathcal{U}(110,\,500)\ \text{ms}, \qquad s_i = \min\Bigl(0{,}985,\ c(g) - 0{,}035\,|Z_i| + 0{,}035\,\frac{\kappa_i - 110}{390}\Bigr), \quad Z_i \sim \mathcal{N}(0,1)
$$

$$
h_i \sim \mathcal{U}(0{,}022,\,0{,}032), \quad [\,s_i - h_i\,;\ \min(1,\ s_i + h_i)\,], \qquad e_i = s_i - \gamma_i, \quad \gamma_i \sim \mathcal{U}(0{,}008,\,0{,}020)
$$

Le parent de chaque individu est tiré au hasard dans le front de la génération précédente (ou du front de départ). Les gènes simulés suivent des lois uniformes : seuil 𝒰(0,3 ; 0,7), tau 𝒰(2 ; 10), lambda 𝒰(0,5 ; 2,5), budget d'activité 𝒰(0,02 ; 0,30), rayon des stimuli 𝒰(4 ; 14) mm, tours ∈ {1, 2}, 3 à 6 contrôles actifs.

Reproductibilité : le générateur PCG est initialisé par la graine, le numéro de campagne c et la génération g ; recalculer une génération donne exactement le même résultat.

$$
\text{PCG}\bigl(\text{graine} \oplus c \times \texttt{0x9E3779B97F4A7C15},\ g + 1\bigr)
$$

## Réalisabilité

Un individu est réalisable si la borne basse de sa sensibilité atteint le plancher diminué de la marge :

$$
R_g = \Bigl\{\, i \;:\; s_i - h_i \;\geq\; \pi - \frac{\delta}{100} \,\Bigr\}
$$

Première génération réalisable possible, estimée dans « Effet de vos réglages » : le meilleur individu dépasse le plafond d'au plus 0,035 et sa borne basse est au moins 0,022 plus bas, d'où 0,80 + 0,035 − 0,022 = 0,813 :

$$
g^{*} = \Bigl\lceil -35 \ln\Bigl(1 - \frac{\pi - \delta/100 - 0{,}813}{0{,}10}\Bigr) \Bigr\rceil - g_0 \quad\text{(jamais si la fraction atteint 1)}
$$

## Front de Pareto

Deux objectifs : sensibilité (à maximiser) et coût (à minimiser). Domination et front :

$$
i \succ j \iff s_i \geq s_j \ \wedge\ \kappa_i \leq \kappa_j \ \wedge\ (s_i > s_j \vee \kappa_i < \kappa_j), \qquad F_g = \{\, i \in R_g : \nexists\, j \in R_g,\ j \succ i \,\}
$$

Calcul : les réalisables sont triés par coût croissant ; un individu entre au front si sa sensibilité dépasse celle de tous les moins coûteux. Le front d'une génération fournit les parents de la suivante.

## Meilleur individu, stagnation et arrêt

Le meilleur d'une génération est le réalisable le plus sensible sur E-val ; le record E de la campagne n'avance que d'au moins 0,1 point :

$$
b_g = \arg\max_{i \in R_g} e_i, \qquad \text{progrès}_g \iff e_{b_g} > E_{g-1} + 0{,}001
$$

$$
E_g = \begin{cases} e_{b_g} & \text{si progrès} \\ E_{g-1} & \text{sinon} \end{cases} \qquad S_g = \begin{cases} 0 & \text{si progrès} \\ S_{g-1} + 1 & \text{sinon (y compris } R_g = \varnothing) \end{cases}
$$

Avant chaque génération, l'orchestrateur arrête la campagne si l'un des trois critères est atteint ; le journal prévient quand il reste 3 générations ou moins avant l'arrêt par stagnation :

$$
\text{arrêt avant } g+1 \iff g \geq G \ \vee\ U_g \geq B \ \vee\ S_g \geq N
$$

## Indicateurs du suivi

Courbes « Progrès par génération » : meilleur sur E-val (absent si R_g est vide), médiane de toute la population sur E-ajust, et écart moyen E-ajust − E-val des réalisables (optimisme de la sélection) :

$$
e_{b_g}, \qquad M_g = \operatorname{med}\{ s_i \}_{i=1}^{P}, \qquad \Gamma_g = \frac{1}{|R_g|} \sum_{i \in R_g} (s_i - e_i)
$$

Fin estimée, « au plus », à partir des 10 dernières générations :

$$
\hat{T} = (G - g) \times \frac{1}{10} \sum_{k=g-9}^{g} d_k
$$

## Images : stockage, fenêtrage et test

À l'import, chaque valeur brute v est rééchelonnée par la pente a et l'ordonnée b du DICOM, puis ramenée sur 16 bits non signés :

$$
u = \operatorname{round}\bigl((a\,v + b - m)\,k\bigr), \qquad m = \min(a v + b), \quad k = \min\Bigl(1,\ \frac{65\,535}{\max(a v + b) - m}\Bigr)
$$

Fenêtre par défaut : celle du DICOM si elle existe, sinon les centiles 1 % et 99 % des pixels non nuls. Affichage d'un pixel pour une fenêtre de centre c et de largeur w (inversé en MONOCHROME1 ou à la demande) :

$$
y = \operatorname{clamp}\Bigl(255\,\frac{u - (c - w/2)}{w},\ 0,\ 255\Bigr), \qquad y_{\text{inversé}} = 255 - y
$$

Sans service d'inférence, test simulé d'un individu (sensibilité s, coût κ) sur une image d'empreinte H : tirage déterministe initialisé par FNV-1a(H ‖ id), donc même image et même individu donnent le même résultat. n zones, placées dans la zone de tissu :

$$
\lambda = \max\Bigl(0,\ \frac{s - 0{,}80}{0{,}15}\Bigr), \qquad n = \mathcal{U}\{0,1,2\} + \mathrm{Bernoulli}(\min(1,\ 0{,}6\,\lambda))
$$

$$
\text{score}_z = \min\bigl(0{,}99,\ 0{,}15 + 0{,}5\,u_1 + 0{,}3\,\lambda\,u_2\bigr), \qquad \text{score}_{\text{image}} = \max_z \text{score}_z, \qquad t = \kappa \times \mathcal{U}(0{,}9,\,1{,}1)
$$

Ces nombres n'ont aucune valeur : ils éprouvent le parcours (import, choix d'un individu, affichage) en attendant le vrai moteur.

Avec le service d'inférence, le CNN reçoit l'image préparée comme à l'entraînement : niveaux ramenés sur 8 bits entre les centiles 0,5 % et 99,5 % (une image déjà sur 8 bits, comme celles de CMMD, est gardée telle quelle), sein placé à gauche, recadrage sur le tissu, réduction sur la toile H × W (1 024 × 832) en gardant les proportions. Score de l'image à partir de la sortie brute z du réseau, et décision au seuil θ choisi à l'entraînement :

$$
p = \sigma(z) = \frac{1}{1 + e^{-z}}, \qquad \text{signalée} \iff p \ge \theta
$$

Zones d'attention (Grad-CAM) : chacune des K cartes A_k de la dernière couche de convolution (32 × 26 cases pour une toile de 1 024 × 832) est pondérée par le gradient moyen du score brut :

$$
\alpha_k = \frac{1}{|\Omega|} \sum_{(i,j) \in \Omega} \frac{\partial z}{\partial A_k^{ij}}, \qquad L = \operatorname{ReLU}\Bigl(\sum_{k=1}^{K} \alpha_k A_k\Bigr)
$$

Les cases hors du sein sont ignorées ; L est divisée par son maximum, les cases à 0,5 ou plus sont regroupées par voisinage (4-connexité), et les trois groupes au plus fort maximum deviennent des zones, ramenées aux pixels de l'image d'origine (réduction, recadrage et retournement défaits). Leur intensité est ce maximum, entre 0,5 et 1.

## CNN : perte, AUC et seuil

Perte d'entraînement (entropie croisée binaire pondérée) sur N images d'étiquette y (1 : malin, 0 : bénin), les malins repondérés par w pour compenser leur proportion :

$$
\mathcal{L} = -\frac{1}{N} \sum_{n=1}^{N} \Bigl[ w\, y_n \log \sigma(z_n) + (1 - y_n) \log\bigl(1 - \sigma(z_n)\bigr) \Bigr], \qquad w = \frac{n_{\text{bénins}}}{n_{\text{malins}}}
$$

La perte affichée pour une époque est la moyenne de ℒ sur toutes les images d'entraînement de cette époque. Elle ne porte que sur ces images : seule l'AUC sur E-ajust, calculée sur des patientes que le CNN n'apprend pas, dit s'il généralise. Sur le graphique, son axe est zoomé sur les valeurs atteintes.

Score d'un sein : moyenne des scores de ses vues. AUC : probabilité qu'un sein malin tiré au hasard ait un score plus haut qu'un sein bénin tiré au hasard (égalités comptées pour moitié), sur les ensembles M des seins malins et B des seins bénins :

$$
\mathrm{AUC} = \frac{1}{|M|\,|B|} \sum_{m \in M} \sum_{b \in B} \Bigl( \mathbb{1}[p_m > p_b] + \tfrac{1}{2}\,\mathbb{1}[p_m = p_b] \Bigr)
$$

La version gardée est celle de la meilleure AUC par sein sur E-ajust ; l'entraînement s'arrête après P époques sans nouveau record (patience). Seuil de décision, fixé ensuite sur E-ajust pour la sensibilité cible s* (0,85 par défaut) : le plus haut seuil qui l'atteint, donc la meilleure spécificité à cette sensibilité.

$$
\theta = \max\bigl\{\, t \;:\; \mathrm{Se}_{\text{E-ajust}}(t) \ge s^{*} \,\bigr\}
$$

## Modèle prévu pour la vraie boucle

Ce qui suit vient de la note de conception ; le socle n'en simule que la forme.

Sensibilité : moyenne sur une fenêtre de ±1 point autour du taux de rappel cible r, mesurée sur E-ajust avec des sorties hors pli, repondérée à la prévalence du dépistage. Le seuil de décision est fixé pour que le taux de rappel repondéré égale celui du programme visé.

$$
s = \frac{1}{0{,}02}\int_{r-0{,}01}^{r+0{,}01} \mathrm{Se}(\rho)\, d\rho
$$

Contrainte de non-infériorité : la borne basse est un quantile bas α, fixé d'avance, de la sensibilité recalculée sur des rééchantillons bootstrap des patientes, au point de fonctionnement ; la même règle vaut par type de lésion face au modèle graine, et pour le fabricant le moins favorable.

$$
s_{\text{bas}} = Q_{\alpha}\bigl(\{ s^{(b)} \}_{b=1}^{B_{\text{boot}}}\bigr) \;\geq\; \pi - \delta
$$

Sélection NSGA-II : tri en fronts non dominés, puis distance de foule pour garder la diversité (au-delà de 3 objectifs, NSGA-III et des points de référence). Pour un individu i d'un front, trié selon chaque objectif f :

$$
\mathrm{dist}_i = \sum_{f} \frac{f_{i+1} - f_{i-1}}{f_{\max} - f_{\min}} \qquad (\text{extrêmes : } +\infty)
$$

Qualité d'un front : hypervolume, la mesure (λ, Lebesgue) de la région dominée par le front jusqu'à un point de référence ρ. Il sert de récompense au choix adaptatif des opérateurs (bandit manchot, test de rupture de Page-Hinkley) et d'arrêt : hypervolume sur E-val stable pendant 15 générations ; bascule vers les macro-mutations après 8.

$$
\mathrm{HV}(F) = \lambda\Bigl(\, \bigcup_{i \in F} [\,\kappa_i,\ \kappa_\rho\,] \times [\,s_\rho,\ s_i\,] \Bigr)
$$

Raisonnement CephaloMesh : une détection du CNN devient un stimulus de courant égal à la confiance multipliée par un gain ; les neurones voisins du cube (12 × 8 × 4 par sein) sont liés avec un poids qui décroît avec la distance d, de portée λ.

$$
I = \text{confiance}_{\text{CNN}} \times \text{gain}, \qquad W(d) = e^{-d/\lambda}
$$

Boucle lente du CNN : paliers asynchrones (ASHA) — un pli partiel, un pli, puis les 5 plis — où seuls les meilleurs montent ; l'aptitude d'un enfant CNN est la moyenne de ses q = 5 meilleurs couples, réévalués sur un autre pli. Optimisme attendu du gagnant parmi ~20 000 configurations : 2 à 3 points de sensibilité (estimation de la note).
