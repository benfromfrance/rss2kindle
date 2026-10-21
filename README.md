# RSS to Kindle 📚

**RSS to Kindle** est une application moderne et légère inspirée de [KindleEar](https://github.com/cdhigh/KindleEar), conçue pour :
1. **Lire vos flux RSS directement depuis votre navigateur** avec une interface épurée à 3 colonnes optimisée typographie papier/encre.
2. **Générer automatiquement chaque jour à heure fixe (par défaut 01:00)** un livre électronique au format **EPUB** haute compatibilité Kindle et l'envoyer à votre adresse `@kindle.com` par email SMTP.
3. **Composer des recueils sur mesure** grâce à un **Panier Kindle** : sélectionnez des articles à la volée, organisez les chapitres, sauvegardez-les en collections ou envoyez-les immédiatement à votre liseuse.

---

## Fonctionnalités clés

- **Lecteur RSS web intégré** :
  - Mise en page à 3 colonnes (Arborescence flux/catégories → Liste des articles → Vue de lecture zen).
  - Vues intelligentes : *Non lus*, *Tous*, *Aujourd'hui*, *Favoris ⭐*, *Pas encore envoyés au Kindle 📦*.
  - Mode lecture assaini avec extraction du **texte intégral à la volée** (Readability).
  - Raccourcis clavier façon Feedly : `j/k` (naviguer), `s` (favori), `m` (lu/non lu), `x` (ajouter/retirer du panier Kindle).
  - Recherche plein texte instantanée (SQLite FTS5).
- **Panier & Envois à la carte** :
  - Sélectionnez des articles de différentes sources pour former un recueil unique.
  - Réordonnez les chapitres par glisser/déposer ou boutons.
  - Choix entre *Envoyer au Kindle par email* ou *Télécharger l'EPUB seul*.
  - Enregistrement de sélections sous forme de **Collections**.
- **Livraison quotidienne programmée (01:00)** :
  - Planificateur cron avec prise en compte du fuseau horaire (ex: `Europe/Paris`).
  - N'envoie que les articles non encore expédiés pour éviter les doublons.
  - Option pour exclure les articles déjà lus dans l'interface web.
  - Option pour ignorer l'envoi si aucun nouvel article n'a été publié.
- **EPUB optimisé pour Amazon Kindle** :
  - Conforme aux exigences strictes de conversion Amazon Send-to-Kindle (EPUB 3, NCX, déclaration de langue, zéro JavaScript/SVG incompatible).
  - Couverture haute résolution générée automatiquement (titre, date en français, nombre d'articles, sources).
  - Sommaire interactif avec regroupement par flux.
  - Traitement des images pour écrans e-ink : redimensionnement automatique, compression JPEG et conversion en niveaux de gris.
- **Gestion des abonnements & OPML** :
  - Découverte automatique des flux depuis l'URL d'un site web.
  - Prévisualisation instantanée des articles avant enregistrement.
  - Import et export au format standard **OPML**.
- **Déploiement zéro dépendance** :
  - Un **seul binaire Go autonome** intégrant l'interface React compilée.
  - Base de données SQLite embarquée en Go pur (sans CGO).

---

## Démarrage rapide

### Option 1 : Avec Docker Compose (recommandé)

```bash
docker compose up -d
```

L'application est accessible sur : **http://localhost:8080**
- Mot de passe par défaut : `admin`

Un serveur SMTP de test (**Mailpit**) est également inclus sur **http://localhost:8025** pour visualiser immédiatement les emails et télécharger les fichiers EPUB envoyés sans polluer votre liseuse.

### Option 2 : Lancement local sans Docker

Nécessite **Go 1.23+** et **Node.js 20+** :

```bash
# 1. Compiler le frontend et le backend
make build

# 2. Lancer l'application
make run
```

---

## Configuration Send-to-Kindle (Amazon)

Pour que vos livres arrivent bien sur votre liseuse Kindle :

1. Rendez-vous sur votre compte Amazon dans la section **Contenu et appareils** → **Préférences** → **Paramètres des documents personnels**.
2. Notez votre **adresse email Kindle** (ex: `votre-nom_12@kindle.com`).
3. Dans la section **Liste des adresses e-mail approuvées pour l'envoi de documents personnels**, ajoutez l'adresse email que vous utilisez comme expéditeur (ex: votre adresse Gmail ou votre domaine).
4. Dans RSS to Kindle, ouvrez la page **Paramètres** :
   - Renseignez l'adresse email Kindle destinataire.
   - Renseignez votre adresse expéditeur approuvée.
   - Saisissez les paramètres de votre serveur SMTP (Gmail, Mailgun, OVH, Brevo, etc.). Pour Gmail, utilisez un *mot de passe d'application*.
   - Cliquez sur **« Envoyer un email de test SMTP »** pour valider la connexion.

---

## Variables d'environnement

| Variable | Défaut | Description |
|---|---|---|
| `R2K_ADDR` | `:8080` | Port et adresse d'écoute HTTP du serveur |
| `R2K_DATA_DIR` | `./data` | Dossier de stockage de la base SQLite et des fichiers EPUB |
| `R2K_ADMIN_PASSWORD` | `admin` | Mot de passe de l'interface d'administration |
| `R2K_SECRET` | *(auto-généré)* | Clé hexadécimale de 32 octets (64 caractères) pour le chiffrement des mots de passe SMTP et la signature des cookies de session |
| `R2K_DEV_CORS_ORIGIN` | `""` | Origine autorisée pour le serveur de développement Vite (ex: `http://localhost:3000`) |

---

## Structure du projet

```
rss2kindle/
├── backend/
│   ├── cmd/rss2kindle/main.go        # Point d'entrée de l'application
│   ├── internal/
│   │   ├── api/                      # Routes Chi, handlers REST, auth cookie
│   │   ├── config/                   # Chargement des variables d'environnement
│   │   ├── delivery/                 # Pipeline et file d'attente des livraisons
│   │   ├── ebook/                    # Générateur EPUB et couverture e-ink
│   │   ├── extract/                  # Readability plein texte & assainissement HTML
│   │   ├── feed/                     # Fetcher RSS/Atom/JSON avec conditional GET
│   │   ├── imageproc/                # Redimensionnement et conversion niveaux de gris
│   │   ├── mailer/                   # Envoi SMTP avec pièces jointes EPUB (go-mail)
│   │   ├── refresher/                # Rafraîchisseur périodique en tâche de fond
│   │   ├── scheduler/                # Cron de planification du digest (robfig/cron)
│   │   └── store/                    # SQLite, migrations, FTS5 et crypto AES-GCM
│   └── web/dist/                     # Frontend compilé embarqué (embed.FS)
├── frontend/                         # Application React 19 + TypeScript + Tailwind CSS v4
│   ├── src/
│   │   ├── components/               # Lecteur 3 colonnes, Panier Kindle, Flux, Paramètres...
│   │   ├── context/                  # AuthContext, CartContext
│   │   └── lib/                      # Client API et helpers
│   └── vite.config.ts
├── Dockerfile                        # Multi-stage build autonome
├── docker-compose.yml                # Configuration prête à l'emploi avec Mailpit
└── Makefile                          # Commandes usuelles
```

---

## Licence

Ce projet est sous licence [MIT](LICENSE).

