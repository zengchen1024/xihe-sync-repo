package dispatcher

import (
	"errors"
	"path/filepath"
)

type Config struct {
	// AccessEndpoint is used to send back the message.
	AccessEndpoint string `json:"access_endpoint"  required:"true"`
	AccessHmac     string `json:"access_hmac"      required:"true"`

	Topic     string `json:"topic"                 required:"true"`
	UserAgent string `json:"user_agent"            required:"true"`

	Workspace string `json:"workspace"             required:"true"`

	// The unit is Gbyte
	SizeOfWorspace int `json:"size_of_workspace"   required:"true"`

	// The unit is Gbyte
	AverageRepoSize int `json:"average_repo_size"  required:"true"`
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

	if !filepath.IsAbs(cfg.Workspace) {
		return errors.New("workspace must be a absolute path")
	}

	if cfg.concurrentSize() <= 0 {
		return errors.New("the concurrent size <= 0")
	}

	return nil
}
