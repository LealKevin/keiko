package config

import (
	"fmt"
	"os"
	"sync"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
	"go.yaml.in/yaml/v3"
)

type UserConfig struct {
	LoopInterval int   `mapstructure:"loop_interval" yaml:"loop_interval"`
	JLPTLevel    []int `mapstructure:"jlpt_level" yaml:"jlpt_level"`

	IsFuriganaVisible    bool `mapstructure:"is_furigana_visible" yaml:"is_furigana_visible"`
	IsJLPTLevelVisible   bool `mapstructure:"is_jlpt_level_visible" yaml:"is_jlpt_level_visible"`
	IsTranslationVisible bool `mapstructure:"is_translation_visible" yaml:"is_translation_visible"`

	AnkiDeck        string `mapstructure:"anki_deck" yaml:"anki_deck"`
	AnkiModeEnabled bool   `mapstructure:"anki_mode_enabled" yaml:"anki_mode_enabled"`
}

type Config struct {
	FilePath   string
	Viper      *viper.Viper
	UserConfig UserConfig
	Updated    chan bool
	mu         sync.RWMutex
}

func New(filePath string) (*Config, error) {
	return &Config{
		FilePath: filePath,
		Viper:    viper.New(),
		Updated:  make(chan bool, 1),
	}, nil
}

func (c *Config) Init() error {
	wd, _ := os.Getwd()
	fmt.Println("Working directory:", wd)
	fmt.Println("Config file:", c.FilePath)

	c.Viper.SetConfigFile(c.FilePath)
	c.Viper.SetConfigType("yaml")

	c.Viper.SetDefault("loop_interval", 10)
	c.Viper.SetDefault("jlpt_level", []int{1, 2, 3, 4, 5})
	c.Viper.SetDefault("is_furigana_visible", true)
	c.Viper.SetDefault("is_jlpt_level_visible", true)
	c.Viper.SetDefault("is_translation_visible", true)
	c.Viper.SetDefault("anki_deck", "")
	c.Viper.SetDefault("anki_mode_enabled", false)

	err := c.Viper.ReadInConfig()
	if err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			return c.Viper.SafeWriteConfig()
		}
	}

	if err := c.Viper.Unmarshal(&c.UserConfig); err != nil {
		return fmt.Errorf("error unmarshalling config: %w", err)
	}

	return nil
}

func (c *Config) Watch() {
	c.Viper.OnConfigChange(func(in fsnotify.Event) {
		c.mu.Lock()
		c.Viper.Unmarshal(&c.UserConfig)
		c.mu.Unlock()
		select {
		case c.Updated <- true:
		default:
		}
	})

	c.Viper.WatchConfig()
}

func (c *Config) Save() error {
	c.mu.RLock()
	data, err := yaml.Marshal(&c.UserConfig)
	c.mu.RUnlock()
	if err != nil {
		return err
	}

	return os.WriteFile(c.FilePath, data, 0o644)
}

func (c *Config) DecreaseInterval() {
	c.mu.Lock()
	if c.UserConfig.LoopInterval == 30 {
		c.mu.Unlock()
		return
	}
	if c.UserConfig.LoopInterval >= 60 {
		c.UserConfig.LoopInterval -= 30
	}
	c.mu.Unlock()
	c.Save()
}

func (c *Config) IncreaseInterval() {
	c.mu.Lock()
	if c.UserConfig.LoopInterval < 3600 {
		c.UserConfig.LoopInterval += 30
	}
	c.mu.Unlock()
	c.Save()
}

func (c *Config) ToggleFurigana() {
	c.mu.Lock()
	c.UserConfig.IsFuriganaVisible = !c.UserConfig.IsFuriganaVisible
	c.mu.Unlock()
	c.Save()
}

func (c *Config) ToggleJLPTLevel() {
	c.mu.Lock()
	c.UserConfig.IsJLPTLevelVisible = !c.UserConfig.IsJLPTLevelVisible
	c.mu.Unlock()
	c.Save()
}

func (c *Config) ToggleTranslation() {
	c.mu.Lock()
	c.UserConfig.IsTranslationVisible = !c.UserConfig.IsTranslationVisible
	c.mu.Unlock()
	c.Save()
}
