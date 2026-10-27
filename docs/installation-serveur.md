# Installer MammoMesh sur un nouveau serveur

`scripts/installer-serveur.sh` installe tout, du système nu à la pile en service, en détectant le matériel. Il est **reprenable** : relancé, il ne refait que ce qui manque. `--simulation` montre chaque commande sans rien changer.

> Outil de recherche : aucun résultat ne sert à un diagnostic.

## Prérequis

- **Système** : Ubuntu 22.04 / 24.04 ou Debian 12 / 13 (et leurs dérivés), amd64 ou arm64.
- **Compte** : un compte ordinaire avec `sudo` (pas root). Sur un serveur neuf :
  `adduser mammomesh && usermod -aG sudo mammomesh && su - mammomesh`.
- **Disque** : 25 Go libres au minimum (pile ~10 Go, PyTorch ~8 Go, 10 Go de marge), plus la banque d'images (23 Go pour CMMD, plusieurs centaines de Go pour le reste, idéalement sur un disque à part).
- **Réseau** : accès à Internet (paquets, images Docker, PyTorch) ; un nom sur le réseau local pour le site.
- **Accès au dépôt** (privé) : clé SSH ou jeton GitHub du compte.

## Installation

```bash
git clone https://github.com/olegrand1976/mammomesh.git   # ou git@github.com:olegrand1976/mammomesh.git
cd mammomesh
./scripts/installer-serveur.sh --simulation     # facultatif : voir ce qui sera fait
./scripts/installer-serveur.sh                  # deux questions : nom du site, adresse d'écoute (et la banque)
sudo reboot                                     # seulement si le script le demande (pilote GPU, groupes)
./scripts/installer-serveur.sh --oui            # après redémarrage : vérification, ne refait que ce qui manque
```

### Options

| Option | Effet |
|---|---|
| `--oui` | Aucune question : valeurs détectées ou par défaut. |
| `--simulation` | Affiche les commandes sans rien modifier. |
| `--site NOM` | Nom du site (`SITE_ADDRESS`), par défaut `<nom de la machine>.lan`. |
| `--ip ADRESSE` | Adresse d'écoute (`BIND_ADDRESS`), par défaut l'adresse de la machine sur le réseau local. |
| `--profils obs,backup` | Profils en plus de ceux du matériel : Prometheus + Grafana, sauvegarde restic. |
| `--accel nvidia\|amd\|intel\|cpu` | Force l'accélérateur au lieu de la détection. |
| `--banque CMMD\|CBIS-DDSM\|NLBS\|tout` | Télécharge la banque à la fin ; `--banque-dir DOSSIER` pour un autre disque. |
| `--sans-pilote` | N'installe pas de pilote GPU (le signale seulement). |
| `--sans-moteur` | Pas d'environnement Python d'entraînement (serveur d'interface seul). |
| `--sans-services` | Pas de démarrage automatique. |
| `TORCH_INDEX=URL` | Index pip de PyTorch imposé (variante CUDA, ROCm, XPU ou CPU). |

## Ce que fait le script

| Étape | Détail |
|---|---|
| 1. Système | Distribution, architecture, CPU, RAM, disque (refus sous 25 Go), sudo, accès Internet. |
| 2. Paquets de base | git, curl, openssl, pciutils, unzip, p7zip, Python 3 avec venv. |
| 3. Matériel | GPU lus dans `/sys/bus/pci` : NVIDIA, AMD, Intel ; choix de l'accélérateur d'entraînement (NVIDIA > AMD > Intel > CPU). |
| 4. Docker | Docker Engine et compose depuis le dépôt officiel, démarrage avec la machine, compte ajouté au groupe `docker`. |
| 5. GPU | **NVIDIA** : pilote (Ubuntu : `ubuntu-drivers`, pilote de calcul sur un serveur sans bureau ; Debian : `nvidia-driver` si `non-free` est actif), `nvidia-container-toolkit`, runtime `nvidia` déclaré à Docker. **AMD** : groupes `render` et `video` ; ROCm signalé s'il manque (installation manuelle, voir plus bas). **Intel** : groupes `render` et `video` (inférence OpenVINO). |
| 6. Système | `vm.overcommit_memory=1` pour Redis. |
| 7. Configuration | `.env` (créé depuis `.env.example` ou complété), secrets (`scripts/init-secrets.sh`), dossier des demandes de l'interface, nom du site dans `/etc/hosts` du serveur. |
| 8. Moteur | `engine/.venv` avec la variante de PyTorch qui correspond au matériel (tableau ci-dessous), puis `pip install -e 'engine[cnn]'` ; affiche le plan matériel (`mammomesh_engine.materiel`). |
| 9. Pile | `scripts/demarrer-pile.sh --build` (profils du matériel), attente de `https://SITE/api/health`, export du certificat racine `mammomesh-root.crt`. |
| 10. Démarrage automatique | `linger`, services utilisateur `mammomesh-pile` (pile au démarrage) et `mammomesh-entrainement` (reprise des entraînements quand le GPU est libre). |
| 11. Banque | Téléchargement facultatif (`examples/telecharger.sh`, espace contrôlé avant). |

### Variante de PyTorch

| Matériel détecté | Variante installée |
|---|---|
| NVIDIA, pilote CUDA ≥ 13.0 (ou pilote tout juste installé) | PyPI par défaut (CUDA 13) |
| NVIDIA, CUDA 12.8 à 12.9 | `whl/cu128` |
| NVIDIA, CUDA 12.6 à 12.7 | `whl/cu126` |
| NVIDIA, pilote plus ancien | CPU, avec un avertissement : mettre le pilote à jour |
| AMD avec ROCm | `whl/rocm6.4` |
| Intel | CPU + OpenVINO |
| Aucun GPU | CPU |

## Après l'installation

1. **Postes clients** : ajouter `ADRESSE SITE` à leur fichier hosts (ou au DNS local), puis importer `mammomesh-root.crt` dans leur magasin de certificats (navigateur ; pour Claude Code : `NODE_EXTRA_CA_CERTS`).
2. **Première connexion** : `https://SITE/login`, compte `admin`, mot de passe de `secrets/admin_password.txt` (`admin$1` par défaut), à changer tout de suite.
3. **Comptes** : voir « Démarrer la pile » dans le README (`migrate user create`).
4. **Banque** : `BANQUE_DIR=/disque/banque ./examples/telecharger.sh --banque CMMD` (voir [jeux d'entraînement](jeux-entrainement.md)).
5. **Entraînement** : depuis l'interface (menu Entraînements, « Nouvel entraînement »), ou
   `cd engine && .venv/bin/python -m mammomesh_engine.cnn --banque /disque/banque`.
6. **Sauvegardes** (profil `backup`) : `BACKUP_DIR` sur un disque ou un NAS hors du serveur ; restauration : [deploy/backup](../deploy/backup/README.md).

## Vérifier

```bash
docker compose ps                                     # tous les services « running » ou « healthy »
systemctl --user status mammomesh-pile mammomesh-entrainement
cd engine && .venv/bin/python -m mammomesh_engine.materiel   # matériel vu et plan d'exécution
./scripts/detecter-materiel.sh                        # profils compose (nvidia, amd, intel, cpu)
```

## Problèmes courants

| Symptôme | Cause et remède |
|---|---|
| Le script demande un redémarrage | Pilote GPU installé ou nouveaux groupes : `sudo reboot`, puis relancer le script. |
| `permission denied` sur le socket Docker | Compte ajouté au groupe `docker` après l'ouverture de la session : redémarrer (ou se reconnecter). |
| `Profils compose : cpu` alors qu'il y a un GPU NVIDIA | Runtime `nvidia` absent de Docker : `sudo nvidia-ctk runtime configure --runtime=docker && sudo systemctl restart docker`. |
| Le site ne répond pas | `docker compose logs api caddy` ; vérifier `SITE_ADDRESS` et `BIND_ADDRESS` dans `.env`, et le pare-feu (port 443). |
| SeaweedFS en lecture seule | Moins de 10 Go libres : libérer de la place. |
| Debian : pas de pilote NVIDIA proposé | Activer `contrib non-free non-free-firmware` dans `/etc/apt/sources.list`, `sudo apt update`, relancer le script. |

## Autres systèmes

Le script ne gère que la famille Debian / Ubuntu. Ailleurs (Fedora, RHEL, Arch…), faire à la main l'équivalent de chaque étape du tableau, puis : `cp .env.example .env` (adapter), `./scripts/init-secrets.sh`, `python3 -m venv engine/.venv` + PyTorch adapté + `pip install -e 'engine[cnn]'`, `./scripts/demarrer-pile.sh --build`, `./scripts/installer-services.sh`.

**AMD (ROCm)** : suivre « Quick start installation » sur rocm.docs.amd.com (paquet `amdgpu-install`, cas d'usage `rocm`), redémarrer, vérifier `/dev/kfd`, puis relancer le script.
