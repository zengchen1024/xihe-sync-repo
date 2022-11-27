package main

import (
	"github.com/opensourceways/community-robot-lib/utils"

	"github.com/opensourceways/xihe-sync-repo/app"
	"github.com/opensourceways/xihe-sync-repo/infrastructure/mysql"
	"github.com/opensourceways/xihe-sync-repo/infrastructure/obsimpl"
	"github.com/opensourceways/xihe-sync-repo/infrastructure/platformimpl"
	"github.com/opensourceways/xihe-sync-repo/syncrepo"
)

type configValidate interface {
	Validate() error
}

type configSetDefault interface {
	SetDefault()
}

type configuration struct {
	App      app.Config          `json:"app"       required:"true"`
	OBS      obsimpl.Config      `json:"obs"       required:"true"`
	Mysql    mysql.Config        `json:"mysql"     required:"true"`
	Gitlab   platformimpl.Config `json:"gitlab"    required:"true"`
	SyncRepo syncrepo.Config     `json:"syncrepo"  required:"true"`
}

func (cfg *configuration) configItems() []interface{} {
	return []interface{}{
		&cfg.App,
		&cfg.OBS,
		&cfg.Gitlab,
		&cfg.Mysql,
		&cfg.SyncRepo,
	}
}

func (cfg *configuration) validate() error {
	if _, err := utils.BuildRequestBody(cfg, ""); err != nil {
		return err
	}

	items := cfg.configItems()

	for _, i := range items {
		if v, ok := i.(configValidate); ok {
			if err := v.Validate(); err != nil {
				return err
			}
		}
	}

	return nil
}

func (cfg *configuration) setDefault() {
	items := cfg.configItems()

	for _, i := range items {
		if v, ok := i.(configSetDefault); ok {
			v.SetDefault()
		}
	}
}

func loadConfig(file string) (cfg configuration, err error) {
	if err = utils.LoadFromYaml(file, &cfg); err != nil {
		return
	}

	cfg.setDefault()

	err = cfg.validate()

	return
}
