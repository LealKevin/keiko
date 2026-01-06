package config

import (
	"fmt"
	"os"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
	"go.yaml.in/yaml/v3"
)

type UserConfig struct {
	LoopInterval int   `mapstructure:"loop_interval" yaml:"loop_interval"`
	JLPTLevel    []int `mapstructure:"jlpt_level" yaml:"jlpt_level"`
}

type Config struct {
	FilePath   string
	Viper      *viper.Viper
	UserConfig UserConfig
	Updated    chan bool
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

	err := c.Viper.ReadInConfig()
	if err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			return c.Viper.SafeWriteConfig()
		}
	}

	err = c.Viper.Unmarshal(&c.UserConfig)
	if err != nil {
		fmt.Println("Error unmarshalling config", err)
	}

	return c.Viper.Unmarshal(&c.UserConfig)
}

func (c *Config) Watch() {
	c.Viper.OnConfigChange(func(in fsnotify.Event) {
		c.Viper.Unmarshal(&c.UserConfig)
		select {
		case c.Updated <- true:
		default:
		}
	})

	c.Viper.WatchConfig()
}

func (c *Config) Save() error {
	data, err := yaml.Marshal(&c.UserConfig)
	if err != nil {
		return err
	}

	return os.WriteFile(c.FilePath, data, 0o644)
}
