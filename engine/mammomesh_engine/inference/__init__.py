"""Inférence du CNN entraîné, au service de l'interface (test d'une image depuis « Images »).

    python -m mammomesh_engine.inference --modeles ../modeles            # sert http://127.0.0.1:8765

GET  /sante                                  modèle chargé, appareil, seuil
POST /predire?lignes=H&colonnes=W&inverser=0&bits=16  corps : pixels uint16 petit-boutiste (gzip accepté)
     → score de malignité de l'image (0-1), seuil choisi à l'entraînement, zones d'attention (Grad-CAM)

Les zones disent où le CNN a regardé pour donner son score : ce ne sont pas des détections de lésions.
Outil de recherche : aucun résultat ne sert à un diagnostic.
"""
