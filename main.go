package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/Scalingo/go-handlers"
	"github.com/Scalingo/go-utils/logger"
	"github.com/kelseyhightower/envconfig"
	"github.com/pkg/errors"

	"database/sql"

	_ "github.com/mattn/go-sqlite3"
)

type Config struct {
	Port int `envconfig:"PORT" default:"5000"`
}

func newConfig() (*Config, error) {
	var cfg Config
	err := envconfig.Process("", &cfg)
	if err != nil {
		return nil, errors.Wrapf(err, "fail to build config from env")
	}
	return &cfg, nil
}

func main() {
	log := logger.Default()
	log.Info("Initializing app")
	cfg, err := newConfig()
	if err != nil {

		log.WithError(err).Error("Fail to initialize configuration")
		os.Exit(1)
	}

	log.Info("Initializing routes")
	router := handlers.NewRouter(log)
	router.HandleFunc("/ping", pongHandler)
	router.HandleFunc("/repos", repositoriesHandler)

	// TODO add /stats (GET)

	log = log.WithField("port", cfg.Port)
	log.Info("Listening...")
	err = http.ListenAndServe(fmt.Sprintf(":%d", cfg.Port), router)
	if err != nil {
		log.WithError(err).Error("Fail to listen to the given port")
		os.Exit(2)
	}
}

func pongHandler(w http.ResponseWriter, r *http.Request, _ map[string]string) error {
	log := logger.Get(r.Context())
	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	err := json.NewEncoder(w).Encode(map[string]string{"status": "pong"})
	if err != nil {
		log.WithError(err).Error("Fail to encode JSON")
	}
	return nil
}

func repositoriesHandler(w http.ResponseWriter, r *http.Request, _ map[string]string) error {
	log := logger.Get(r.Context())

	db, err := sql.Open("sqlite3", "data.db")
	if err != nil {
		log.WithError(err).Error("Fail to open database")
		return err
	}
	defer db.Close()

	// Build sql query
	query := "SELECT * FROM repositories WHERE 1=1"
	var args []interface{}

	if name := r.URL.Query().Get("name"); name != "" {
		query += " AND name LIKE ?"
		args = append(args, "%"+name+"%")
	}
	if stars := r.URL.Query().Get("stars"); stars != "" {
		query += " AND stars >= ?"
		args = append(args, stars)
	}
	if archived := r.URL.Query().Get("archived"); archived != "" {
		query += " AND archived = ?"
		args = append(args, archived)
	}

	rows, err := db.Query(query, args...)
	if err != nil {
		log.WithError(err).Error("Fail to execute query")
		return err
	}
	defer rows.Close()

	// Assuming your table has columns like id, name, stars, archived
	var repos []map[string]interface{}
	for rows.Next() {
		var id int
		var name string
		var stars int
		var archived bool
		if err := rows.Scan(&id, &name, &stars, &archived); err != nil {
			log.WithError(err).Error("Fail to scan row")
			continue
		}
		repos = append(repos, map[string]interface{}{
			"id":       id,
			"name":     name,
			"stars":    stars,
			"archived": archived,
		})
	}

	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	// Example: respond with a list of repositories
	// You can replace this with the actual logic to fetch or display repositories
	w.WriteHeader(http.StatusOK)
	err = json.NewEncoder(w).Encode(repos)
	if err != nil {
		log.WithError(err).Error("Fail to encode JSON")
	}
	return nil
}
