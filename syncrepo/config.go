package syncrepo

import (
	"errors"

	kfklib "github.com/opensourceways/kafka-lib/agent"
)

type Config struct {
	Topic     string `json:"topic"                 required:"true"`
	UserAgent string `json:"user_agent"            required:"true"`

	// The unit is Gbyte
	SizeOfWorspace int `json:"size_of_workspace"   required:"true"`

	// The unit is Gbyte
	AverageRepoSize int `json:"average_repo_size"  required:"true"`

	// kfk
	Kafka kfklib.Config `json:"kafka"`
}

func (cfg *Config) concurrentSize() int {
	return cfg.SizeOfWorspace / (cfg.AverageRepoSize) / 2
}

func (cfg *Config) Validate() error {
	if cfg.Topic == "" {
		return errors.New("missing topic")
	}

	if cfg.UserAgent == "" {
		return errors.New("missing user_agent")
	}

	if cfg.AverageRepoSize <= 0 {
		return errors.New("invalid average_repo_size")
	}

	if cfg.concurrentSize() <= 0 {
		return errors.New("the concurrent size <= 0")
	}

	return cfg.Kafka.Validate()
}
