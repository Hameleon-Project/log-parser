package storage

import (
	"database/sql"
	"log"
	"os"
	"path/filepath"

	_ "github.com/lib/pq"
)

func NewPostgresStorage(dsn string) (*sql.DB, error) {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		return nil, err
	}

	log.Println("database connection established")

	schema, pathUsed, err := readSchemaFile()
	if err != nil {
		log.Printf("warning: could not read schema.sql: %v", err)
	} else {
		if _, err := db.Exec(string(schema)); err != nil {
			log.Printf("error applying schema: %v", err)
		} else {
			log.Printf("database schema applied from %s", pathUsed)
		}
	}

	return db, nil
}

func readSchemaFile() ([]byte, string, error) {
	candidates := []string{"schema.sql", "./schema.sql"}
	if exe, err := os.Executable(); err == nil {
		dir := filepath.Dir(exe)
		candidates = append(candidates, filepath.Join(dir, "schema.sql"))
	}
	for _, p := range candidates {
		b, err := os.ReadFile(p)
		if err == nil {
			return b, p, nil
		}
	}
	return nil, "", os.ErrNotExist
}
