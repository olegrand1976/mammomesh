"""Premier CNN de MammoMesh (jalon 1, étape 1) : bénin ou malin sur CMMD.

    python -m mammomesh_engine.cnn --banque ../banque          # index, prétraitement, entraînement, évaluation

Étapes, chacune reprenable :
1. cmmd         : index des images étiquetées (une étiquette par sein) et découpage par patiente
                  en entraînement / ajustement (E-ajust) / validation (E-val) ;
2. pretraitement: DICOM → PNG 8 bits orienté, recadré et mis à la taille d'entrée (cache local),
                  après contrôle de la place libre ;
3. entrainement : EfficientNet-B0, réglé par le plan matériel (appareil, précision, lot mesuré),
                  arrêt sur l'AUC par sein en ajustement, mesures finales en validation.

Outil de recherche : aucun résultat ne sert à un diagnostic.
"""
