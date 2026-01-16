package db

import (
	"testing"

	"github.com/LealKevin/keiko/internal/data"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestDB(t *testing.T) *DB {
	db, err := Open(":memory:")
	require.NoError(t, err)
	require.NoError(t, db.Migrate())
	return db
}

func seedTestWords(t *testing.T, db *DB) {
	words := []data.Word{
		{Word: "犬", Meaning: "dog", Furigana: "いぬ", Romaji: "inu", Level: 5},
		{Word: "猫", Meaning: "cat", Furigana: "ねこ", Romaji: "neko", Level: 5},
		{Word: "食べる", Meaning: "to eat", Furigana: "たべる", Romaji: "taberu", Level: 4},
		{Word: "飲む", Meaning: "to drink", Furigana: "のむ", Romaji: "nomu", Level: 4},
		{Word: "経済", Meaning: "economy", Furigana: "けいざい", Romaji: "keizai", Level: 2},
	}
	require.NoError(t, db.SeedVocab(words))
}

func TestOpen(t *testing.T) {
	db, err := Open(":memory:")

	require.NoError(t, err)
	assert.NotNil(t, db)

	err = db.Close()
	assert.NoError(t, err)
}

func TestMigrate(t *testing.T) {
	db, err := Open(":memory:")
	require.NoError(t, err)
	defer db.Close()

	err = db.Migrate()
	assert.NoError(t, err)

	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM words").Scan(&count)
	assert.NoError(t, err)
	assert.Equal(t, 0, count)
}

func TestSeedVocab(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	words := []data.Word{
		{Word: "犬", Meaning: "dog", Furigana: "いぬ", Romaji: "inu", Level: 5},
		{Word: "猫", Meaning: "cat", Furigana: "ねこ", Romaji: "neko", Level: 5},
	}

	err := db.SeedVocab(words)
	assert.NoError(t, err)

	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM words").Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 2, count)
}

func TestGetNextWord(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	seedTestWords(t, db)

	t.Run("returns word from specified levels", func(t *testing.T) {
		word, err := db.GetNextWord([]int{5})

		assert.NoError(t, err)
		assert.Equal(t, 5, word.Level)
		assert.Contains(t, []string{"犬", "猫"}, word.Word)
	})

	t.Run("returns word from multiple levels", func(t *testing.T) {
		word, err := db.GetNextWord([]int{4, 5})

		assert.NoError(t, err)
		assert.Contains(t, []int{4, 5}, word.Level)
	})

	t.Run("returns error when no levels provided", func(t *testing.T) {
		_, err := db.GetNextWord([]int{})

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no levels provided")
	})

	t.Run("returns error when no words found", func(t *testing.T) {
		_, err := db.GetNextWord([]int{1})

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no words found")
	})
}

func TestMarkWordAsSeen(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	seedTestWords(t, db)

	err := db.MarkWordAsSeen(1)
	assert.NoError(t, err)

	var seen int
	err = db.QueryRow("SELECT seen FROM words WHERE id = 1").Scan(&seen)
	require.NoError(t, err)
	assert.Equal(t, 1, seen)
}

func TestResetSeenWords(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	seedTestWords(t, db)

	_, err := db.Exec("UPDATE words SET seen = 1 WHERE level = 5")
	require.NoError(t, err)

	err = db.ResetSeenWords(5)
	assert.NoError(t, err)

	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM words WHERE level = 5 AND seen = 0").Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 2, count)
}

func TestGetWordsCount(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	seedTestWords(t, db)

	t.Run("count single level", func(t *testing.T) {
		count, err := db.GetWordsCount([]int{5})

		assert.NoError(t, err)
		assert.Equal(t, 2, count)
	})

	t.Run("count multiple levels", func(t *testing.T) {
		count, err := db.GetWordsCount([]int{4, 5})

		assert.NoError(t, err)
		assert.Equal(t, 4, count)
	})

	t.Run("count all levels", func(t *testing.T) {
		count, err := db.GetWordsCount([]int{1, 2, 3, 4, 5})

		assert.NoError(t, err)
		assert.Equal(t, 5, count)
	})

	t.Run("count non-existent level", func(t *testing.T) {
		count, err := db.GetWordsCount([]int{1})

		assert.NoError(t, err)
		assert.Equal(t, 0, count)
	})
}

func TestGetNextWordRespectsSeenFlag(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	words := []data.Word{
		{Word: "犬", Meaning: "dog", Furigana: "いぬ", Romaji: "inu", Level: 5},
	}
	require.NoError(t, db.SeedVocab(words))

	word, err := db.GetNextWord([]int{5})
	require.NoError(t, err)
	assert.Equal(t, "犬", word.Word)

	require.NoError(t, db.MarkWordAsSeen(1))

	_, err = db.GetNextWord([]int{5})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no words found")
}

func TestGetNextWordReturnsValidID(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	words := []data.Word{
		{Word: "猫", Meaning: "cat", Furigana: "ねこ", Romaji: "neko", Level: 5},
	}
	require.NoError(t, db.SeedVocab(words))

	word, err := db.GetNextWord([]int{5})
	require.NoError(t, err)
	assert.NotZero(t, word.ID, "GetNextWord should return a word with a valid ID")

	require.NoError(t, db.MarkWordAsSeen(word.ID))

	_, err = db.GetNextWord([]int{5})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no words found")
}
