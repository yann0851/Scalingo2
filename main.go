package main

import (
	"compress/gzip"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
)

type LanguageDetails struct {
	Bytes int `json:"bytes"`
}

type Owner struct {
	Login string `json:"login"`
}

type License struct {
	Name string `json:"name"`
}

type Repository struct {
	FullName       string                     `json:"full_name"`
	Owner          Owner                      `json:"owner"`
	RepositoryName string                     `json:"repository"`
	Languages      map[string]LanguageDetails `json:"languages"`
	License        *License                   `json:"license"`
}

type GitHubSearchResponse struct {
	Items []Repository `json:"items"`
}

type Cache struct {
	mu    sync.Mutex
	store map[string]map[string]LanguageDetails
}

func (c *Cache) Get(key string) (map[string]LanguageDetails, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	langDetails, exists := c.store[key]
	return langDetails, exists
}

func (c *Cache) Set(key string, value map[string]LanguageDetails) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.store[key] = value
}

var repositories []Repository = []Repository{}
var mu sync.Mutex
var cache = Cache{store: make(map[string]map[string]LanguageDetails)}

func fetchRepositories(token string) ([]Repository, error) {
	baseURL := "https://api.github.com/search/repositories"
	query := "?q=stars:>1&sort=stars&order=desc&per_page=100&page=1"
	fullURL := baseURL + query

	var allRepositories []Repository
	for page := 1; len(allRepositories) < 100; page++ {
		fullURL = fmt.Sprintf("%s%s&page=%d", baseURL, query, page)

		req, err := http.NewRequest("GET", fullURL, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("Authorization", "Bearer "+token)

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("GitHub API request failed with status code: %d", resp.StatusCode)
		}

		var searchResponse GitHubSearchResponse
		if err := json.NewDecoder(resp.Body).Decode(&searchResponse); err != nil {
			return nil, err
		}

		for i := range searchResponse.Items {
			repoParts := strings.Split(searchResponse.Items[i].FullName, "/")
			if len(repoParts) == 2 {
				searchResponse.Items[i].RepositoryName = repoParts[1]
			}
		}

		allRepositories = append(allRepositories, searchResponse.Items...)
		if len(searchResponse.Items) == 0 {
			break
		}
	}

	if len(allRepositories) > 100 {
		allRepositories = allRepositories[:100]
	}

	return allRepositories, nil
}

func fetchRepositoryLanguages(repo *Repository, token string) {
	// Vérifier dans le cache avant de faire la requête
	if cachedLangs, exists := cache.Get(repo.FullName); exists {
		repo.Languages = cachedLangs
		return
	}

	url := fmt.Sprintf("https://api.github.com/repos/%s/languages", repo.FullName)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		fmt.Printf("Error creating request for repository %s: %v\n", repo.FullName, err)
		return
	}
	req.Header.Set("Authorization", "Bearer "+token)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("Error fetching languages for repository %s: %v\n", repo.FullName, err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		fmt.Printf("GitHub API request failed for repository %s with status code: %d\n", repo.FullName, resp.StatusCode)
		return
	}

	var languages map[string]int
	if err := json.NewDecoder(resp.Body).Decode(&languages); err != nil {
		fmt.Printf("Error decoding languages for repository %s: %v\n", repo.FullName, err)
		return
	}

	langDetails := make(map[string]LanguageDetails)
	for lang, bytes := range languages {
		langDetails[lang] = LanguageDetails{Bytes: bytes}
	}

	// Stocker les résultats dans le cache
	cache.Set(repo.FullName, langDetails)

	repo.Languages = langDetails
}

func worker(repoChan <-chan *Repository, token string, wg *sync.WaitGroup) {
	defer wg.Done()

	for repo := range repoChan {
		fetchRepositoryLanguages(repo, token)
	}
}

func repositoriesHandler(w http.ResponseWriter, r *http.Request) {
	mu.Lock()
	defer mu.Unlock()

	language := r.URL.Query().Get("language")
	license := r.URL.Query().Get("license")
	minBytesStr := r.URL.Query().Get("min_bytes")
	pageStr := r.URL.Query().Get("page")
	perPageStr := r.URL.Query().Get("per_page")

	var minBytes int
	if minBytesStr != "" {
		fmt.Sscanf(minBytesStr, "%d", &minBytes)
	}

	page := 1
	if pageStr != "" {
		parsedPage, err := strconv.Atoi(pageStr)
		if err == nil && parsedPage > 0 {
			page = parsedPage
		}
	}

	perPage := 10
	if perPageStr != "" {
		parsedPerPage, err := strconv.Atoi(perPageStr)
		if err == nil && parsedPerPage > 0 {
			perPage = parsedPerPage
		}
	}

	var result []Repository
	for _, repo := range repositories {
		// Filtrer par langage si spécifié
		if language != "" {
			bytes, exists := repo.Languages[language]
			if !exists || (minBytes > 0 && bytes.Bytes < minBytes) {
				continue
			}
		}

		// Filtrer par licence si spécifié
		if license != "" {
			if repo.License == nil || repo.License.Name != license {
				continue
			}
		}

		result = append(result, repo)
	}

	// Pagination
	start := (page - 1) * perPage
	end := start + perPage
	if start >= len(result) {
		result = []Repository{}
	} else if end > len(result) {
		result = result[start:]
	} else {
		result = result[start:end]
	}

	response := map[string]interface{}{
		"repositories": result,
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Encoding", "gzip")
	gzipWriter := gzip.NewWriter(w)
	defer gzipWriter.Close()

	json.NewEncoder(gzipWriter).Encode(response)
}

func languagesSummaryHandler(w http.ResponseWriter, r *http.Request) {
	mu.Lock()
	defer mu.Unlock()

	languageSummary := make(map[string]int)
	for _, repo := range repositories {
		for lang, details := range repo.Languages {
			languageSummary[lang] += details.Bytes
		}
	}

	totalBytes := 0
	for _, bytes := range languageSummary {
		totalBytes += bytes
	}

	languagePercentage := make(map[string]float64)
	for lang, bytes := range languageSummary {
		languagePercentage[lang] = (float64(bytes) / float64(totalBytes)) * 100
	}

	response := map[string]interface{}{
		"language_summary":                languageSummary,
		"language_percentage":             languagePercentage,
		"total_repositories_per_language": countRepositoriesPerLanguage(),
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Encoding", "gzip")
	gzipWriter := gzip.NewWriter(w)
	defer gzipWriter.Close()

	json.NewEncoder(gzipWriter).Encode(response)
}

func countRepositoriesPerLanguage() map[string]int {
	languageCount := make(map[string]int)
	for _, repo := range repositories {
		for lang := range repo.Languages {
			languageCount[lang]++
		}
	}
	return languageCount
}

func main() {
	token := "Your_GitHub_Token"

	var err error
	repositories, err = fetchRepositories(token)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	// Canal de dépôt pour les travailleurs
	repoChan := make(chan *Repository, len(repositories))

	var wg sync.WaitGroup

	// Définir le nombre d'ouvriers, par exemple 10
	numWorkers := 10
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go worker(repoChan, token, &wg)
	}

	// Ajouter des dépôts au canal
	for i := range repositories {
		repoChan <- &repositories[i]
	}
	close(repoChan)

	// Attendre que tous les ouvriers aient terminé
	wg.Wait()

	http.HandleFunc("/repositories", repositoriesHandler)
	http.HandleFunc("/languages_summary", languagesSummaryHandler)

	fmt.Println("Server is running on port 8080...")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		fmt.Println("Server failed to start:", err)
	}
}
