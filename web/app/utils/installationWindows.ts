// Installation de la pile sur Windows 11 : source unique de la page Documentation › Installation sur Windows 11,
// de docs/installation-windows-11.md (un test vérifie la concordance) et du PDF qui en est tiré
// (docs/installation-windows-11.pdf, copié dans web/public/docs pour le téléchargement).
import type { Partie } from '~/utils/fonctionnement'

export const etatWindows = 'État au 27 octobre 2026'
export const pdfWindows = '/docs/installation-windows-11.pdf'

export const partiesWindows: Partie[] = [
  {
    id: 'principe',
    titre: 'Principe',
    blocs: [
      { p: 'Sur Windows 11, MammoMesh tourne dans WSL 2 (Windows Subsystem for Linux), sous Debian : la même pile qu\'un serveur Linux, installée par le même script (scripts/installer-serveur.sh), qui reconnaît WSL. Docker Engine est installé dans Debian (pas Docker Desktop) ; le GPU NVIDIA est prêté par le pilote Windows. Outil de recherche : aucun résultat ne sert à un diagnostic.' },
      {
        table: {
          entetes: ['Élément', 'Où', 'Rôle'],
          lignes: [
            ['WSL 2 + Debian', 'Windows', 'Machine Linux intégrée, démarrée à l\'ouverture de session.'],
            ['Pilote NVIDIA', 'Windows seulement', 'Prête le GPU à Debian (CUDA). Jamais de pilote NVIDIA Linux dans WSL.'],
            ['Docker Engine, compose, nvidia-container-toolkit', 'Debian', 'Font tourner la pile (base, API, interface, Caddy…) et donnent le GPU aux conteneurs.'],
            ['Moteur Python (engine/.venv)', 'Debian', 'Entraînement du CNN sur le GPU, veilleur de reprise.'],
            ['Navigateur', 'Windows', 'Ouvre https://localhost ou le nom du site.'],
          ],
        },
      },
    ],
  },
  {
    id: 'prerequis',
    titre: 'Prérequis',
    blocs: [
      {
        points: [
          'Windows 11 22H2 ou plus récent (réseau « mirrored »), à jour, avec un compte administrateur.',
          'Virtualisation activée dans le BIOS/UEFI (Intel VT-x ou AMD-V) ; le Gestionnaire des tâches l\'indique (Performances › Processeur › Virtualisation).',
          '16 Go de RAM conseillés (8 au minimum), 100 Go libres sur le disque qui accueillera Debian, plus la banque d\'images (23 Go pour CMMD).',
          'Facultatif : GPU NVIDIA avec un pilote Windows récent (GeForce Game Ready, Studio ou RTX Enterprise), pour entraîner le CNN.',
          'Accès au dépôt GitHub privé de MammoMesh (clé SSH ou jeton).',
        ],
      },
    ],
  },
  {
    id: 'wsl',
    titre: '1. Installer WSL et Debian',
    blocs: [
      { p: 'Dans PowerShell ouvert en administrateur :' },
      { code: 'wsl --install -d Debian\n# redémarrer Windows si demandé, puis :\nwsl --update\nwsl --set-default-version 2\nwsl -l -v          # Debian doit apparaître en VERSION 2', langage: 'powershell' },
      { p: 'Au premier lancement de Debian (menu Démarrer), choisir un nom d\'utilisateur Linux et son mot de passe : c\'est le compte qui fera tourner MammoMesh (avec sudo).' },
      { p: 'Banque volumineuse ou disque C: trop petit : déplacer Debian sur un autre disque avant d\'aller plus loin.' },
      { code: 'wsl --shutdown\nwsl --manage Debian --move D:\\WSL\\Debian', langage: 'powershell' },
    ],
  },
  {
    id: 'wslconfig',
    titre: '2. Régler WSL (mémoire, réseau)',
    blocs: [
      { p: 'Créer le fichier %UserProfile%\\.wslconfig (par exemple C:\\Users\\olivier\\.wslconfig) ; adapter la mémoire et les processeurs à la machine :' },
      { code: '[wsl2]\nmemory=12GB\nprocessors=8\nswap=8GB\n# Le poste et WSL partagent les mêmes adresses : la pile est joignable depuis le réseau local.\nnetworkingMode=mirrored', langage: 'ini' },
      { p: 'Puis, dans PowerShell en administrateur, autoriser les connexions entrantes vers WSL sur les ports 80 et 443 (pare-feu Hyper-V), et redémarrer WSL :' },
      { code: 'New-NetFirewallHyperVRule -Name MammoMesh -DisplayName "MammoMesh (HTTPS)" -Direction Inbound `\n  -VMCreatorId \'{40E0AC32-46A5-438A-A0B2-2B479E8F2E90}\' -Protocol TCP -LocalPorts 80,443\nwsl --shutdown', langage: 'powershell' },
      { p: 'Sans réseau « mirrored » (Windows plus ancien), la pile reste joignable depuis le poste lui-même par https://localhost ; pour le réseau local, ajouter un renvoi de port (netsh interface portproxy) vers l\'adresse donnée par « wsl hostname -I ».' },
    ],
  },
  {
    id: 'gpu',
    titre: '3. GPU NVIDIA (facultatif)',
    blocs: [
      {
        points: [
          'Installer ou mettre à jour le pilote NVIDIA pour Windows (site de NVIDIA ou GeForce Experience). C\'est le seul pilote à installer.',
          'Ne jamais installer de pilote NVIDIA ni le paquet « cuda » complet dans Debian : ils casseraient l\'accès au GPU prêté par Windows.',
          'Après installation, redémarrer WSL (wsl --shutdown), puis vérifier dans Debian :',
        ],
      },
      { code: '/usr/lib/wsl/lib/nvidia-smi       # doit afficher la carte et la version CUDA', langage: 'bash' },
      { p: 'Sous WSL, nvidia-smi ne liste pas les programmes qui utilisent le GPU : le veilleur des entraînements juge alors la disponibilité du GPU sur sa mémoire libre seulement.' },
    ],
  },
  {
    id: 'code',
    titre: '4. Récupérer MammoMesh dans Debian',
    blocs: [
      { p: 'Dans le terminal Debian, dans le dossier personnel Linux (et non sous /mnt/c, beaucoup plus lent) :' },
      { code: 'sudo apt update && sudo apt install -y git\ncd ~\n# HTTPS (jeton GitHub) ; en SSH : git clone git@github.com:olegrand1976/mammomesh.git\ngit clone https://github.com/olegrand1976/mammomesh.git\ncd mammomesh', langage: 'bash' },
      { p: 'Dépôt privé : pour HTTPS, un jeton GitHub tient lieu de mot de passe ; pour SSH, créer une clé (ssh-keygen -t ed25519) et l\'ajouter au compte GitHub.' },
    ],
  },
  {
    id: 'installation',
    titre: '5. Lancer l\'installation',
    blocs: [
      { code: './scripts/installer-serveur.sh --simulation   # facultatif : voir ce qui sera fait\n./scripts/installer-serveur.sh --site mammomesh.lan', langage: 'bash' },
      { p: 'Le script reconnaît WSL. Il active systemd s\'il ne l\'est pas : il s\'arrête alors en demandant « wsl --shutdown » dans PowerShell ; rouvrir Debian et relancer la même commande. Il installe ensuite :' },
      {
        points: [
          'les paquets utiles (git, curl, openssl, pciutils, unzip, p7zip, Python 3 avec venv) ;',
          'Docker Engine et docker compose depuis le dépôt officiel de Docker, démarrés avec Debian ;',
          'si un GPU NVIDIA est prêté par Windows : nvidia-container-toolkit et le runtime nvidia de Docker, sans pilote Linux ;',
          'la configuration (.env, secrets), le moteur Python avec PyTorch adapté (CUDA si GPU, sinon CPU) ;',
          'la pile, construite et vérifiée, le certificat racine du site (mammomesh-root.crt) et les services de démarrage automatique.',
        ],
      },
      { p: 'Fin d\'installation : si le script le demande (nouveau groupe docker), redémarrer WSL avec « wsl --shutdown » dans PowerShell, rouvrir Debian, puis relancer le script avec --oui pour vérifier.' },
    ],
  },
  {
    id: 'acces',
    titre: '6. Ouvrir l\'interface depuis Windows',
    blocs: [
      {
        etapes: [
          'Fichier hosts de Windows (Bloc-notes lancé en administrateur) : ouvrir C:\\Windows\\System32\\drivers\\etc\\hosts et ajouter la ligne « 127.0.0.1 mammomesh.lan » (ou l\'adresse du poste sur le réseau, pour les autres postes).',
          'Certificat : dans l\'Explorateur, ouvrir \\\\wsl$\\Debian\\home\\VOTRE_NOM\\mammomesh, double-cliquer mammomesh-root.crt, « Installer le certificat », Ordinateur local, magasin « Autorités de certification racines de confiance ». Firefox : Paramètres › Vie privée et sécurité › Certificats › Importer.',
          'Ouvrir https://mammomesh.lan (ou https://localhost), se connecter avec le compte admin et le mot de passe de secrets/admin_password.txt, puis le changer.',
        ],
      },
      { code: 'certutil -addstore -f Root "\\\\wsl$\\Debian\\home\\VOTRE_NOM\\mammomesh\\mammomesh-root.crt"', langage: 'powershell' },
    ],
  },
  {
    id: 'demarrage',
    titre: '7. Démarrage automatique avec Windows',
    blocs: [
      { p: 'Dans Debian, Docker relance la pile et les services utilisateur (pile, veilleur des entraînements) démarrent avec systemd. Encore faut-il que Debian démarre, et reste allumé : une tâche planifiée Windows s\'en charge à l\'ouverture de session (PowerShell) :' },
      { code: '$action = New-ScheduledTaskAction -Execute "wsl.exe" `\n  -Argument "-d Debian --exec sleep infinity"\n$declencheur = New-ScheduledTaskTrigger -AtLogOn\n$reglages = New-ScheduledTaskSettingsSet -ExecutionTimeLimit ([TimeSpan]::Zero) `\n  -AllowStartIfOnBatteries -DontStopIfGoingOnBatteries\nRegister-ScheduledTask -TaskName "MammoMesh (WSL)" -Action $action `\n  -Trigger $declencheur -Settings $reglages `\n  -Description "Démarre Debian (WSL), donc la pile MammoMesh, et le garde allumé"', langage: 'powershell' },
      { p: 'Vérifier après une nouvelle session Windows : https://mammomesh.lan répond, et dans Debian « systemctl --user status mammomesh-pile mammomesh-entrainement ».' },
    ],
  },
  {
    id: 'banque',
    titre: '8. Banque d\'images et entraînement',
    blocs: [
      { code: '# dans Debian, depuis ~/mammomesh (l\'espace libre est contrôlé avant tout téléchargement)\n./examples/telecharger.sh --banque CMMD', langage: 'bash' },
      { p: 'Ensuite, dans l\'interface : menu Entraînements, « Nouvel entraînement ». Les gros jeux (CBIS-DDSM, NLBS) demandent plusieurs centaines de Go : Debian sur un disque dédié (étape 1), ou un NAS monté dans Debian.' },
    ],
  },
  {
    id: 'depannage',
    titre: 'Dépannage',
    blocs: [
      {
        table: {
          entetes: ['Symptôme', 'Cause et remède'],
          lignes: [
            ['« wsl --install » échoue', 'Virtualisation désactivée dans le BIOS, ou fonctionnalités « Plateforme de machine virtuelle » et « Sous-système Windows pour Linux » absentes : les activer, redémarrer.'],
            ['Le script demande « wsl --shutdown »', 'systemd vient d\'être activé (/etc/wsl.conf) : wsl --shutdown dans PowerShell, rouvrir Debian, relancer le script.'],
            ['permission denied sur le socket Docker', 'Compte ajouté au groupe docker : wsl --shutdown, rouvrir Debian.'],
            ['nvidia-smi introuvable ou sans GPU', 'Pilote NVIDIA Windows absent ou ancien : le mettre à jour sous Windows, puis wsl --shutdown. Ne rien installer côté Debian.'],
            ['Le site ne répond pas depuis Windows', 'Ligne manquante dans le fichier hosts de Windows ; pile arrêtée (docker compose ps dans Debian) ; pare-feu Hyper-V (étape 2).'],
            ['Le site ne répond pas depuis un autre poste', 'Réseau « mirrored » non actif (.wslconfig, puis wsl --shutdown) ou règle de pare-feu manquante ; hosts du poste client vers l\'adresse du poste Windows.'],
            ['Debian s\'arrête quand on ferme le terminal', 'Tâche planifiée absente ou désactivée (étape 7).'],
            ['Disque C: plein', 'Déplacer Debian (wsl --manage Debian --move) ; la place libérée dans Debian ne revient à Windows qu\'après compactage du disque virtuel.'],
          ],
        },
      },
    ],
  },
  {
    id: 'desinstaller',
    titre: 'Désinstaller',
    blocs: [
      { code: 'Unregister-ScheduledTask -TaskName "MammoMesh (WSL)" -Confirm:$false\nRemove-NetFirewallHyperVRule -Name MammoMesh\n# Efface Debian et tout ce qu\'il contient (pile, base, banque) :\nwsl --unregister Debian', langage: 'powershell' },
    ],
  },
]
