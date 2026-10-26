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

## Tests

`make exemples` lance le script. `go test -count=1 ./internal/imagerie -run TestExemples` (dans `backend/`) importe chaque fichier GitHub présent et vérifie le résultat attendu du manifeste ; les séries TCIA présentes doivent être acceptées comme MG. Sans fichiers, le test est ignoré.

Sans téléchargement, l'application fournit des mammographies synthétiques : liens « Exemples synthétiques » de l'écran Images, ou `GET /api/images/exemple.dcm`.
