package service

import (
	"testing"

	"github.com/LealKevin/keiko/internal/data"
	"github.com/LealKevin/keiko/internal/db"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestService(t *testing.T) (VocabService, *db.DB) {
	database, err := db.Open(":memory:")
	require.NoError(t, err)
	require.NoError(t, database.Migrate())

	words := []data.Word{
		{Word: "犬", Meaning: "dog", Furigana: "いぬ", Romaji: "inu", Level: 5},
		{Word: "猫", Meaning: "cat", Furigana: "ねこ", Romaji: "neko", Level: 5},
		{Word: "食べる", Meaning: "to eat", Furigana: "たべる", Romaji: "taberu", Level: 4},
	}
	require.NoError(t, database.SeedVocab(words))

	svc := New(database)
	return svc, database
}

func TestServiceGetNextWord(t *testing.T) {
	t.Run("returns word from requested levels", func(t *testing.T) {
		svc, database := setupTestService(t)
		defer database.Close()

		word, err := svc.GetNextWord([]int{5})

		assert.NoError(t, err)
		assert.Equal(t, 5, word.Level)
		assert.Contains(t, []string{"犬", "猫"}, word.Word)
	})

	t.Run("returns error when no words match level", func(t *testing.T) {
		database, err := db.Open(":memory:")
		require.NoError(t, err)
		defer database.Close()
		require.NoError(t, database.Migrate())

		words := []data.Word{
			{Word: "犬", Meaning: "dog", Furigana: "いぬ", Romaji: "inu", Level: 5},
		}
		require.NoError(t, database.SeedVocab(words))

		svc := New(database)

		_, err = svc.GetNextWord([]int{1})
		assert.Error(t, err)
	})

	t.Run("returns error when db is empty", func(t *testing.T) {
		database, err := db.Open(":memory:")
		require.NoError(t, err)
		defer database.Close()
		require.NoError(t, database.Migrate())

		svc := New(database)

		_, err = svc.GetNextWord([]int{5})
		assert.Error(t, err)
	})
}

func TestServiceMarkWordAsSeen(t *testing.T) {
	svc, database := setupTestService(t)
	defer database.Close()

	err := svc.MarkWordAsSeen(1)

	assert.NoError(t, err)

	var seen int
	err = database.QueryRow("SELECT seen FROM words WHERE id = 1").Scan(&seen)
	require.NoError(t, err)
	assert.Equal(t, 1, seen)
}

func TestServiceResetSeenWords(t *testing.T) {
	svc, database := setupTestService(t)
	defer database.Close()

	_, _ = svc.GetNextWord([]int{5})
	_, _ = svc.GetNextWord([]int{5})

	err := svc.ResetSeenWords(5)
	assert.NoError(t, err)

	var count int
	err = database.QueryRow("SELECT COUNT(*) FROM words WHERE level = 5 AND seen = 0").Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 2, count)
}

func TestServiceGetWordsCount(t *testing.T) {
	svc, database := setupTestService(t)
	defer database.Close()

	count, err := svc.GetWordsCount([]int{4, 5})

	assert.NoError(t, err)
	assert.Equal(t, 3, count)
}

func TestCheckIfAllWordsSeen(t *testing.T) {
	t.Run("returns false when words exist", func(t *testing.T) {
		database, err := db.Open(":memory:")
		require.NoError(t, err)
		defer database.Close()
		require.NoError(t, database.Migrate())

		words := []data.Word{
			{Word: "犬", Meaning: "dog", Furigana: "いぬ", Romaji: "inu", Level: 5},
		}
		require.NoError(t, database.SeedVocab(words))

		svc := New(database).(*service)
		result := svc.CheckIfAllWordsSeen([]int{5})
		assert.False(t, result)
	})

	t.Run("returns true when no words for level", func(t *testing.T) {
		database, err := db.Open(":memory:")
		require.NoError(t, err)
		defer database.Close()
		require.NoError(t, database.Migrate())

		svc := New(database).(*service)
		result := svc.CheckIfAllWordsSeen([]int{5})
		assert.True(t, result)
	})
}
