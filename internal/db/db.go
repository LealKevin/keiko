package db

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/LealKevin/keiko/internal/data"
	_ "github.com/mattn/go-sqlite3"
)

type DB struct {
	*sql.DB
}

func Open(path string) (*DB, error) {
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, err
	}
	return &DB{db}, nil
}

func (db *DB) Close() error {
	return db.DB.Close()
}

func (db *DB) Migrate() error {
	_, err := db.Exec(`
	    	CREATE TABLE IF NOT EXISTS words (
					id INTEGER PRIMARY KEY AUTOINCREMENT,
	    		word TEXT,
	    		meaning TEXT,
	    		furigana TEXT,
	    		romaji TEXT,
	    		level INTEGER,
					seen INTEGER DEFAULT 0
	    	)
	`)
	return err
}

func (db *DB) SeedVocab(words []data.Word) error {
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("error beginning transaction: %s", err)
	}

	defer tx.Rollback()

	statement, err := tx.Prepare(`
		INSERT INTO words (word, meaning, furigana, romaji, level)
		VALUES (?, ?, ?, ?, ?)
		`)
	if err != nil {
		return fmt.Errorf("error preparing statement: %s", err)
	}

	defer statement.Close()

	for _, word := range words {
		_, err = statement.Exec(word.Word, word.Meaning, word.Furigana, word.Romaji, word.Level)
		if err != nil {
			return fmt.Errorf("error inserting word: %s", err)
		}
	}
	err = tx.Commit()
	if err != nil {
		return fmt.Errorf("error commiting transaction: %s", err)
	}

	return nil
}

func (db *DB) GetNextWord(levels []int) (data.Word, error) {
	if len(levels) == 0 {
		return data.Word{}, fmt.Errorf("no levels provided")
	}

	placeholders := make([]string, len(levels))
	args := make([]interface{}, len(levels))

	for i, level := range levels {
		placeholders[i] = "?"
		args[i] = level
	}

	query := fmt.Sprintf(`
		SELECT word, meaning, furigana, romaji, level 
		FROM words 
		WHERE seen = 0 AND level IN (%s) 
		ORDER BY RANDOM() 
		LIMIT 1`,
		strings.Join(placeholders, ", "),
	)

	var word data.Word
	err := db.QueryRow(query, args...).Scan(
		&word.Word,
		&word.Meaning,
		&word.Furigana,
		&word.Romaji,
		&word.Level,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return data.Word{}, fmt.Errorf("no words found")
		}

		return data.Word{}, fmt.Errorf("error fetching word: %s", err)
	}

	return word, nil
}

func (db *DB) MarkWordAsSeen(id int) error {
	_, err := db.Exec(`
		UPDATE words
		SET seen = 1
		WHERE id = ?`,
		id,
	)
	if err != nil {
		return fmt.Errorf("error marking word as seen: %s", err)
	}
	return nil
}

func (db *DB) ResetSeenWords(level int) error {
	_, err := db.Exec(`
		UPDATE words
		SET seen = 0
		WHERE level = ?`,
		level)
	if err != nil {
		return fmt.Errorf("error resetting seen words: %s", err)
	}
	return nil
}

func (db *DB) GetWordsCount(levels []int) (int, error) {
	var count int
	placeholder := make([]string, len(levels))
	args := make([]interface{}, len(levels))

	for _, level := range levels {
		placeholder = append(placeholder, "?")
		args[level] = level
	}

	query := fmt.Sprintf(`
		SELECT COUNT(*) 
		FROM words
		WHERE level IN (%s)`, strings.Join(placeholder, ", "))

	err := db.QueryRow(query, args...).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("error getting words count: %s", err)
	}
	return count, nil
}
