# README.md

## Description du Projet

Ce projet est une API qui interagit avec l'API de GitHub pour rechercher et récupérer des informations sur des dépôts populaires, notamment leurs langages, licences, et autres métadonnées. L'API offre des fonctionnalités de filtrage, de statistiques et de mise en cache pour améliorer l'expérience utilisateur et optimiser les performances.

## Architecture du Projet

Le projet est structuré en utilisant les concepts suivants :

### Technologies Utilisées
- **Go (Golang)** : Langage principal du projet, choisi pour sa capacité à gérer la concurrence de manière performante et simple grâce aux goroutines.
- **API GitHub** : Pour rechercher les dépôts et récupérer les informations sur leurs langages et licences.
- **HTTP avec Gzip** : Pour compresser les réponses JSON et minimiser la bande passante utilisée.

### Décisions d'Architecture
- **Routines Go (Goroutines)** : Des goroutines sont utilisées pour paralléliser la récupération des informations de langages des dépôts. Cela permet de réduire le temps nécessaire pour traiter un grand nombre de requêtes réseau, en utilisant efficacement la concurrence native de Go.
- **Worker Pool** : Un pool de workers est utilisé pour limiter le nombre de goroutines simultanées, afin d'éviter une surcharge du système et d'assurer la stabilité de l'application.
- **Mise en Cache** : La mise en cache des langages des dépôts réduit le nombre de requêtes vers l'API GitHub, ce qui améliore la performance globale de l'application et réduit la consommation de l'API.
- **Pagination** : Pour limiter le nombre de dépôts renvoyés par requête, la pagination est implémentée afin de rendre les requêtes plus performantes et permettre une meilleure gestion des données côté client.
- **Compression Gzip** : Toutes les réponses sont compressées avec Gzip afin de réduire la taille des données transférées et d'améliorer la vitesse de transfert.

## Documentation de l'API

### Endpoints Disponibles

#### 1. **GET /repositories**
- **Description** : Récupère une liste de dépôts avec des options de filtrage et de pagination.
- **Paramètres Acceptés** :
  - `language` (string, optionnel) : Filtrer les dépôts par un langage spécifique.
  - `license` (string, optionnel) : Filtrer les dépôts par une licence spécifique.
  - `min_bytes` (int, optionnel) : Filtrer les dépôts par le nombre minimal de bytes de code dans un langage spécifique.
  - `page` (int, optionnel) : Numéro de la page pour la pagination (par défaut : 1).
  - `per_page` (int, optionnel) : Nombre de dépôts par page (par défaut : 10).
- **Format de Réponse** :
  - JSON compressé avec Gzip.
  - Contient une liste des dépôts et leurs informations telles que `full_name`, `owner`, `repository`, `languages`, et `license`.
- **Exemple de Requête** :
  ```
  GET /repositories?language=Go&per_page=5&page=2
  ```
- **Exemple de Réponse** :
  ```json
  {
    "repositories": [
      {
        "full_name": "golang/go",
        "owner": { "login": "golang" },
        "repository": "go",
        "languages": {
          "Go": { "bytes": 45056646 }
        },
        "license": { "name": "BSD-3-Clause" }
      }
    ]
  }
  ```

#### 2. **GET /languages_summary**
- **Description** : Récupère un résumé des langages utilisés parmi les dépôts analysés, y compris le nombre total de bytes par langage et le pourcentage de chaque langage.
- **Format de Réponse** :
  - JSON compressé avec Gzip.
  - Contient le `language_summary` (nombre total de bytes par langage), `language_percentage` (pourcentage d'utilisation de chaque langage), et `total_repositories_per_language` (nombre de dépôts par langage).
- **Exemple de Requête** :
  ```
  GET /languages_summary
  ```
- **Exemple de Réponse** :
  ```json
  {
    "language_summary": {
      "Go": 12000000,
      "JavaScript": 15000000
    },
    "language_percentage": {
      "Go": 44.44,
      "JavaScript": 55.56
    },
    "total_repositories_per_language": {
      "Go": 50,
      "JavaScript": 70
    }
  }
  ```

## Exécution du Projet

### Prérequis
- **Go 1.17+** : Installez Go pour compiler et exécuter le projet.
- **Token GitHub** : Un token GitHub personnel est nécessaire pour accéder à l'API GitHub sans limitation stricte.

### Instructions d'Installation
1. Clonez le dépôt :
   ```sh
   git clone
   ```
2. Accédez au répertoire du projet :
   ```sh
   ```
3. Ajoutez votre token GitHub dans le fichier `main.go` à la place de `YOUR_GITHUB_TOKEN`.

### Lancer le Serveur
- Pour lancer le serveur, exécutez la commande suivante :
  ```sh
  go run main.go
  ```
- Le serveur écoute par défaut sur le port `8080`.

