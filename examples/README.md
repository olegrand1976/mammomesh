# DICOM d'exemple

Fichiers DICOM de sources libres pour tester l'import et la visionneuse (écran **Images**). Outil de recherche : aucun de ces fichiers ne sert à un diagnostic.

Les `.dcm` ne sont pas versionnés. Pour les récupérer :

```sh
./examples/telecharger.sh               # GitHub (~22 Mo) puis TCIA (~110 Mo)
./examples/telecharger.sh --sans-tcia   # GitHub seulement
./examples/telecharger.sh --tcia-seulement
```

Le script lit `manifeste.json`, vérifie l'empreinte SHA-256 de chaque fichier GitHub (URL figées sur un commit) et ne retélécharge pas ce qui est déjà là. Il demande `curl`, `python3` et, pour TCIA, `unzip`.

## Fichiers GitHub (`github/`)

Huit fichiers courts qui couvrent les cas acceptés et refusés par le lecteur.

| Fichier | Contenu | Résultat attendu | Source, licence |
|---|---|---|---|
| `fo-dicom_MG_Siemens_CC_R.dcm` | Mammographie réelle, Siemens Mammomat Inspiration, CC droite, 2082 × 2800, 12 bits | Acceptée (MG, CC, R) | [fo-dicom](https://github.com/fo-dicom/fo-dicom), MS-PL |
| `pydicom_RG3_CR_MONOCHROME1.dcm` | Radiographie CR 1760 × 1760, MONOCHROME1 | Acceptée, affichage inversé | [pydicom-data](https://github.com/pydicom/pydicom-data), MIT (NEMA WG04) |
| `pydicom_CT_small_signe.dcm` | Coupe CT 128 × 128, pixels signés | Acceptée | [pydicom](https://github.com/pydicom/pydicom), MIT (NEMA WG04) |
| `pydicom_MR_small_big_endian.dcm` | IRM 64 × 64, Explicit VR Big Endian | Acceptée | pydicom, MIT (NEMA WG04) |
| `pydicom_RG1_JPEG2000.dcm` | Radiographie en JPEG 2000 | Refusée : JPEG 2000 | pydicom-data, MIT |
| `pydicom_JPEG_lossless.dcm` | JPEG sans perte (processus 14) | Refusée : JPEG Lossless | pydicom-data, MIT |
| `pydicom_JPEG_12bits.dcm` | JPEG avec perte sur 12 bits | Refusée : JPEG Extended | pydicom, MIT ; données GDCM, BSD |
| `pydicom_SC_couleur.dcm` | Capture secondaire RGB | Refusée : couleur | pydicom-data, MIT |

La mammographie fo-dicom n'a pas d'origine documentée ; ses en-têtes ont été anonymisés (noms et établissement fictifs). Elle sert aux essais techniques seulement.

## Séries TCIA (`tcia/`)

Mammographies de collections publiques de [The Cancer Imaging Archive](https://www.cancerimagingarchive.net), récupérées par l'API publique NBIA (sans compte).

| Collection | Patiente | Contenu | Licence |
|---|---|---|---|
| CMMD | D1-0001 | 2 images (~8,8 Mo) | CC BY 4.0 |
| CMMD | D2-0140 | 4 images, CC et MLO des deux seins (~17,6 Mo) | CC BY 4.0 |
| CBIS-DDSM | Mass-Training_P_00001_LEFT_CC | Film numérisé, masse, image complète (~29 Mo) | CC BY 3.0 |
| CBIS-DDSM | Mass-Training_P_00001_LEFT_MLO | Même patiente, MLO (~25 Mo) | CC BY 3.0 |
| CBIS-DDSM | Calc-Training_P_00005_RIGHT_CC | Microcalcifications, image complète (~28 Mo) | CC BY 3.0 |

Les fichiers sont nommés `tcia/<collection>/<patiente>_<n>.dcm`. Pour CBIS-DDSM, seule l'image complète est gardée (ni découpes ni masques).

**Citations obligatoires (CC BY)** :

- Cui C., Li L., Cai H. et al. *The Chinese Mammography Database (CMMD)*. The Cancer Imaging Archive, 2021. doi:[10.7937/tcia.eqde-4b16](https://doi.org/10.7937/tcia.eqde-4b16)
- Sawyer-Lee R., Gimenez F., Hoogi A., Rubin D. *Curated Breast Imaging Subset of DDSM (CBIS-DDSM)*. The Cancer Imaging Archive, 2016. doi:[10.7937/K9/TCIA.2016.7O02S9CY](https://doi.org/10.7937/K9/TCIA.2016.7O02S9CY)
- Clark K. et al. *The Cancer Imaging Archive (TCIA): Maintaining and Operating a Public Information Repository*. J Digit Imaging 2013;26(6):1045-1057.

Les conditions de TCIA interdisent toute tentative de réidentification des patientes.

## Banque d'entraînement (jalon 0)

Collections complètes, sous licence CC BY (usage commercial permis, citation obligatoire). Elles sont décrites dans la clé `banque` du manifeste. Le choix des jeux, ceux à demander ou écartés, les citations et les sources sont dans [docs/jeux-entrainement.md](../docs/jeux-entrainement.md).

| Nom | Contenu | Volume | Licence |
|---|---|---|---|
| `CBIS-DDSM` | 1 566 patientes, 10 239 images (complètes, découpes, masques ROI), descriptions CSV | ~164 Go | CC BY 3.0 |
| `CMMD` | 1 775 patientes, bénin ou malin prouvé par biopsie, données cliniques XLSX | ~23 Go | CC BY 4.0 |
| `NLBS` | 5 997 examens de dépistage, 26 988 images, 149 cancers, CSV des étiquettes ; sans annotation de lésion | ~350 Go (archives) | CC BY 4.0 |

```sh
BANQUE_DIR=/chemin/disque ./examples/telecharger.sh --banque CMMD    # ou CBIS-DDSM, NLBS, tout
make banque BANQUE=CMMD BANQUE_DIR=/chemin/disque
```

- **Espace disque** : avant chaque collection, le script calcule ce qui reste à télécharger (tailles publiées par TCIA) et refuse de démarrer si le disque ne peut pas tout recevoir en gardant `MARGE_GO` libres (10 par défaut). Il recontrôle avant chaque série et s'arrête proprement si la place vient à manquer. Pour NLBS, il contrôle aussi la taille décompressée avant l'extraction.
- **Reprise** : relancer la même commande. Les séries terminées (marqueur `.ok`, nombre d'images vérifié) sont sautées ; les échecs sont listés dans `tcia/<collection>/.echecs`.
- **Parallélisme** : `PARALLELE` séries à la fois (4 par défaut).
- **NLBS** : FRDR ne publie pas de lien direct. Le script vérifie la place et indique la marche à suivre : transfert Globus (ou « Download as Zip », puis dézipper) des archives `.7z` et du CSV vers `BANQUE_DIR/nlbs/archives`, puis relance pour l'extraction (`7z`, paquet `p7zip-full`). Les archives ne sont pas supprimées.
- **Arborescence** : `tcia/<collection>/<PatientID>/<SeriesInstanceUID>/*.dcm`, `tcia/<collection>/series.json` (liste TCIA), `tcia/<collection>/metadonnees/`, `nlbs/donnees/`.
- Par défaut, `BANQUE_DIR` vaut `./banque` (non versionné). Prévoir un disque dédié.

Citation supplémentaire : *Full Field Digital Mammography Dataset from a Population Screening Program* (NL-Breast-Screen), Scientific Data, 2025, doi:[10.1038/s41597-025-05866-0](https://doi.org/10.1038/s41597-025-05866-0) ; données : FRDR, doi:[10.20383/103.01526](https://doi.org/10.20383/103.01526).

Non retenues pour l'instant (licence non commerciale ou à vérifier) : RSNA-SMBC, EMBED, VinDr-Mammo. À demander plus tard : CSAW-CC (CC BY 4.0, sur demande, ~2,5 To).

## Tests

`make exemples` lance le script. `go test -count=1 ./internal/imagerie -run TestExemples` (dans `backend/`) importe chaque fichier GitHub présent et vérifie le résultat attendu du manifeste ; les séries TCIA présentes doivent être acceptées comme MG. Sans fichiers, le test est ignoré.

Sans téléchargement, l'application fournit des mammographies synthétiques : liens « Exemples synthétiques » de l'écran Images, ou `GET /api/images/exemple.dcm`.
