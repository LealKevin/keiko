package config

import (
	"os"

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
}

func New(filePath string) (*Config, error) {
	return &Config{
		FilePath: filePath,
		Viper:    viper.New(),
	}, nil
}

func (c *Config) Init() error {
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

	return c.Viper.Unmarshal(&c.UserConfig)
}

func (c *Config) Save() error {
	data, err := yaml.Marshal(&c.UserConfig)
	if err != nil {
		return err
	}

	return os.WriteFile(c.FilePath, data, 0o644)
}
