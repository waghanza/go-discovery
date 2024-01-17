package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sync"

	_ "github.com/mattn/go-sqlite3"
)

type ListOf struct {
	URL string `json:"url"`
}

type Repository struct {
	Name          string `json:"name"`
	NumberOfStars int    `json:"stargazers_count"`
	Archived      bool   `json:"archived"`
}

func openDatabase() (*sql.DB, error) {
	db, err := sql.Open("sqlite3", "data.db")
	if err != nil {
		return nil, err
	}

	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS repositories (id INTEGER PRIMARY KEY AUTOINCREMENT, name TEXT, stars INTEGER, archived BOOLEAN)`)
	return db, err
}

func fetchRepositories(client *http.Client, req *http.Request) ([]ListOf, error) {
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("request failed with status code: %d", resp.StatusCode)
	}

	var repos []ListOf
	err = json.NewDecoder(resp.Body).Decode(&repos)
	return repos, err
}

func fetchData(token, url string, wg *sync.WaitGroup, ch chan<- *Repository) {
	defer wg.Done()

	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	resp, err := (&http.Client{}).Do(req)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error fetching data: %v\n", err)
		ch <- nil
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		fmt.Fprintf(os.Stderr, "request failed: %s\n", resp.Status)
		ch <- nil
		return
	}

	var repo Repository
	if err := json.NewDecoder(resp.Body).Decode(&repo); err != nil {
		fmt.Fprintf(os.Stderr, "error decoding json: %v\n", err)
		ch <- nil
		return
	}

	ch <- &repo
}

func main() {
	if len(os.Args) <= 1 {
		fmt.Fprintln(os.Stderr, "missing token")
		return
	}

	token, apiURL := os.Args[1], "https://api.github.com/repositories" // TODO must be lastest
	req, _ := http.NewRequest("GET", apiURL, nil)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	repos, err := fetchRepositories(&http.Client{}, req)
	if err != nil {
		fmt.Fprintf(os.Stderr, "fetch error: %v\n", err)
		return
	}

	var wg sync.WaitGroup
	ch := make(chan *Repository, len(repos))

	for _, r := range repos {
		wg.Add(1)
		go fetchData(token, r.URL, &wg, ch)
	}

	wg.Wait()
	close(ch)

	db, err := openDatabase()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error opening database: %v\n", err)
		return
	}
	defer db.Close()

	tx, _ := db.Begin()
	stmt, _ := tx.Prepare(`INSERT INTO repositories (name, stars, archived) VALUES (?, ?, ?)`)
	for repo := range ch {
		if repo != nil {
			_, _ = stmt.Exec(repo.Name, repo.NumberOfStars, repo.Archived)
		}
	}
	stmt.Close()
	tx.Commit()
}
