package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"

	_ "github.com/mattn/go-sqlite3"
)

func main() {
	db, err := OpenDB()
	if err != nil {
		fmt.Println(err)
		return
	}
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	mux := http.NewServeMux()
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("./static"))))
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		IndexHandler(w, r, db)
	})
	mux.HandleFunc("/save-score", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			// Parse the JSON data from the request body
			var data struct {
				Nickname string `json:"nickname"`
				Score    int    `json:"score"`
			}
			if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			// Insert the data into the database
			_, err := db.Exec("INSERT INTO scores (username, score) VALUES (?, ?)", data.Nickname, data.Score)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			// Respond with a success status code

		} else {
			http.NotFound(w, r)
		}
	})
	mux.HandleFunc("/get-score", func(w http.ResponseWriter, r *http.Request) {
		updateScoreboard(w, db)
	})
	fmt.Printf("Listening on port %v\n", port)
	fmt.Println("server started . . .")
	fmt.Println("ctrl(cmd) + click: http://localhost:8080/")
	http.ListenAndServe(":"+port, mux)
	defer db.Close()
}

const createtable string = `
CREATE TABLE IF NOT EXISTS scores (
	score_ID INTEGER PRIMARY KEY AUTOINCREMENT NOT NULL ,
	username TEXT NOT NULL, 
	score INTEGER NOT NULL
);`

func OpenDB() (*sql.DB, error) {
	dbPath := "./database.db"
	// Check if the database file exists
	if _, err := os.Stat(dbPath); errors.Is(err, os.ErrNotExist) {
		// Open a new database connection
		db, err := sql.Open("sqlite3", dbPath)
		if err != nil {
			return nil, err
		}
		// Tables
		if _, err := db.Exec(createtable); err != nil {
			return nil, err
		}
		return db, nil
	}
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, err
	}
	return db, nil
}
func IndexHandler(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	if r.Method == http.MethodGet {
		// Serve the HTML file
		http.ServeFile(w, r, "static/index.html")
	} else {
		// Handle other HTTP methods or routes if needed
		http.NotFound(w, r)
		return
	}
}
func updateScoreboard(w http.ResponseWriter, db *sql.DB) {
	// Query the database to get scores in descending order
	rows, err := db.Query("SELECT username, score FROM scores ORDER BY score DESC LIMIT 10")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var scores []struct {
		Username string `json:"username"`
		Score    int    `json:"score"`
	}

	// Iterate over the rows and populate the scores slice
	for rows.Next() {
		var score struct {
			Username string `json:"username"`
			Score    int    `json:"score"`
		}
		if err := rows.Scan(&score.Username, &score.Score); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		scores = append(scores, score)
	}

	// Respond with the scores in JSON format
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(scores); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		fmt.Print(err)
		return
	}
}
