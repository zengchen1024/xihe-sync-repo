package app

import (
	"errors"
	"path/filepath"
)

type Config struct {
	ServiceConfig

	HelperConfig
}

type ServiceConfig struct {
	WorkDir       string `json:"work_dir"        required:"true"`
	Expiry        int64  `json:"expiry"`
	SyncFileShell string `json:"sync_file_shell" required:"true"`
}

type HelperConfig struct {
	LFSPath    string `json:"lfs_path"    required:"true"`
	RepoPath   string `json:"repo_path"   required:"true"`
	CommitFile string `json:"commit_file" required:"true"`
}

func (c *Config) SetDefault() {
	if c.Expiry <= 0 {
		c.Expiry = 10 * 3600 // seconds
	}
}

func (c *Config) Validate() error {
	if !filepath.IsAbs(c.WorkDir) {
		return errors.New("work_dir must be an absolute path")
	}

	if !filepath.IsAbs(c.SyncFileShell) {
		return errors.New("sync_file_shell must be an absolute path")
	}

	if filepath.IsAbs(c.LFSPath) {
		return errors.New("lfs_path can't start with /")
	}

	if filepath.IsAbs(c.RepoPath) {
		return errors.New("repo_path can't start with /")
	}

	return nil
}
