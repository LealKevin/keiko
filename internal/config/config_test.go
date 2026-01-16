package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestConfig(t *testing.T) (*Config, string) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	cfg, err := New(configPath)
	require.NoError(t, err)
	require.NoError(t, cfg.Init())

	return cfg, configPath
}

func TestNew(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	cfg, err := New(configPath)

	assert.NoError(t, err)
	assert.NotNil(t, cfg)
	assert.Equal(t, configPath, cfg.FilePath)
	assert.NotNil(t, cfg.Viper)
	assert.NotNil(t, cfg.Updated)
}

func TestInit(t *testing.T) {
	t.Run("creates config with defaults", func(t *testing.T) {
		cfg, _ := setupTestConfig(t)

		assert.Equal(t, 10, cfg.UserConfig.LoopInterval)
		assert.Equal(t, []int{1, 2, 3, 4, 5}, cfg.UserConfig.JLPTLevel)
		assert.True(t, cfg.UserConfig.IsFuriganaVisible)
		assert.True(t, cfg.UserConfig.IsJLPTLevelVisible)
		assert.True(t, cfg.UserConfig.IsTranslationVisible)
	})

	t.Run("loads existing config", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "config.yaml")

		content := `loop_interval: 60
jlpt_level: [3, 4, 5]
is_furigana_visible: false
is_jlpt_level_visible: true
is_translation_visible: false
`
		require.NoError(t, os.WriteFile(configPath, []byte(content), 0o644))

		cfg, err := New(configPath)
		require.NoError(t, err)
		require.NoError(t, cfg.Init())

		assert.Equal(t, 60, cfg.UserConfig.LoopInterval)
		assert.Equal(t, []int{3, 4, 5}, cfg.UserConfig.JLPTLevel)
		assert.False(t, cfg.UserConfig.IsFuriganaVisible)
		assert.True(t, cfg.UserConfig.IsJLPTLevelVisible)
		assert.False(t, cfg.UserConfig.IsTranslationVisible)
	})
}

func TestToggleFurigana(t *testing.T) {
	cfg, _ := setupTestConfig(t)

	assert.True(t, cfg.UserConfig.IsFuriganaVisible)

	cfg.ToggleFurigana()
	assert.False(t, cfg.UserConfig.IsFuriganaVisible)

	cfg.ToggleFurigana()
	assert.True(t, cfg.UserConfig.IsFuriganaVisible)
}

func TestToggleJLPTLevel(t *testing.T) {
	cfg, _ := setupTestConfig(t)

	assert.True(t, cfg.UserConfig.IsJLPTLevelVisible)

	cfg.ToggleJLPTLevel()
	assert.False(t, cfg.UserConfig.IsJLPTLevelVisible)

	cfg.ToggleJLPTLevel()
	assert.True(t, cfg.UserConfig.IsJLPTLevelVisible)
}

func TestToggleTranslation(t *testing.T) {
	cfg, _ := setupTestConfig(t)

	assert.True(t, cfg.UserConfig.IsTranslationVisible)

	cfg.ToggleTranslation()
	assert.False(t, cfg.UserConfig.IsTranslationVisible)

	cfg.ToggleTranslation()
	assert.True(t, cfg.UserConfig.IsTranslationVisible)
}

func TestIncreaseInterval(t *testing.T) {
	cfg, _ := setupTestConfig(t)

	assert.Equal(t, 10, cfg.UserConfig.LoopInterval)

	cfg.IncreaseInterval()
	assert.Equal(t, 40, cfg.UserConfig.LoopInterval)

	cfg.IncreaseInterval()
	assert.Equal(t, 70, cfg.UserConfig.LoopInterval)
}

func TestIncreaseIntervalMax(t *testing.T) {
	cfg, _ := setupTestConfig(t)
	cfg.UserConfig.LoopInterval = 3600

	cfg.IncreaseInterval()

	assert.Equal(t, 3600, cfg.UserConfig.LoopInterval)
}

func TestDecreaseInterval(t *testing.T) {
	cfg, _ := setupTestConfig(t)
	cfg.UserConfig.LoopInterval = 120

	cfg.DecreaseInterval()
	assert.Equal(t, 90, cfg.UserConfig.LoopInterval)

	cfg.DecreaseInterval()
	assert.Equal(t, 60, cfg.UserConfig.LoopInterval)

	cfg.DecreaseInterval()
	assert.Equal(t, 30, cfg.UserConfig.LoopInterval)
}

func TestDecreaseIntervalMin(t *testing.T) {
	cfg, _ := setupTestConfig(t)
	cfg.UserConfig.LoopInterval = 30

	cfg.DecreaseInterval()

	assert.Equal(t, 30, cfg.UserConfig.LoopInterval)
}

func TestSave(t *testing.T) {
	cfg, configPath := setupTestConfig(t)

	cfg.UserConfig.LoopInterval = 120
	cfg.UserConfig.IsFuriganaVisible = false

	err := cfg.Save()
	assert.NoError(t, err)

	content, err := os.ReadFile(configPath)
	require.NoError(t, err)

	assert.Contains(t, string(content), "loop_interval: 120")
	assert.Contains(t, string(content), "is_furigana_visible: false")
}
