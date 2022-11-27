package obsimpl

import (
	"errors"
	"path/filepath"
)

type Config struct {
	OBSUtilPath string `json:"obsutil_path"  required:"true"`
	AccessKey   string `json:"access_key"    required:"true"`
	SecretKey   string `json:"secret_key"    required:"true"`
	Endpoint    string `json:"endpoint"      required:"true"`
	Bucket      string `json:"bucket"        required:"true"`
}

func (c *Config) Validate() error {
	if !filepath.IsAbs(c.OBSUtilPath) {
		return errors.New("obsutil_path must be an absolute path")
	}

	return nil
}
