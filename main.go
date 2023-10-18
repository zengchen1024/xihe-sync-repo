package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/opensourceways/community-robot-lib/logrusutil"
	liboptions "github.com/opensourceways/community-robot-lib/options"
	redislib "github.com/opensourceways/redis-lib"
	"github.com/sirupsen/logrus"

	"github.com/opensourceways/xihe-sync-repo/app"
	"github.com/opensourceways/xihe-sync-repo/infrastructure/messages"
	"github.com/opensourceways/xihe-sync-repo/infrastructure/mysql"
	"github.com/opensourceways/xihe-sync-repo/infrastructure/obsimpl"
	"github.com/opensourceways/xihe-sync-repo/infrastructure/platformimpl"
	"github.com/opensourceways/xihe-sync-repo/infrastructure/synclockimpl"
	"github.com/opensourceways/xihe-sync-repo/syncrepo"
)

type options struct {
	service           liboptions.ServiceOptions
	enableDebug       bool
	kafkamqConfigFile string
}

func (o *options) Validate() error {
	return o.service.Validate()
}

func gatherOptions(fs *flag.FlagSet, args ...string) (options, error) {
	var o options

	o.service.AddFlags(fs)

	fs.StringVar(
		&o.kafkamqConfigFile, "kafkamq-config-file", "/etc/kafkamq/config.yaml",
		"Path to the file containing config of kafkamq.",
	)

	fs.BoolVar(
		&o.enableDebug, "enable_debug", false,
		"whether to enable debug model.",
	)

	err := fs.Parse(args)

	return o, err
}

const component = "xihe-sync-repo"

func main() {
	logrusutil.ComponentInit(component)
	log := logrus.NewEntry(logrus.StandardLogger())

	o, err := gatherOptions(flag.NewFlagSet(os.Args[0], flag.ExitOnError), os.Args[1:]...)

	if err != nil {
		logrus.Fatalf("new options failed, err:%s", err.Error())
	}

	if err := o.Validate(); err != nil {
		log.Errorf("Invalid options, err:%s", err.Error())

		return
	}

	if o.enableDebug {
		logrus.SetLevel(logrus.DebugLevel)
		logrus.Debug("debug enabled.")
	}

	// load config
	cfg, err := loadConfig(o.service.ConfigFile)
	if err != nil {
		log.Errorf("Error loading config, err:%v", err)

		return
	}

	if err := os.Remove(o.service.ConfigFile); err != nil {
		log.Fatalf("Error remove config file, err:%v", err)
	}

	// init redis for kafka
	redisCfg := cfg.getRedisConfig()
	if err = redislib.Init(&redisCfg, true); err != nil {
		log.Errorf("Error init redis of kafka error, err:%v", err)

		return
	}

	defer redislib.Close()

	// init kafka
	kafkaCfg, err := messages.LoadKafkaConfig(o.kafkamqConfigFile)
	if err != nil {
		log.Errorf("Error loading kfk config, err:%v", err)

		return
	}

	if err := os.Remove(o.kafkamqConfigFile); err != nil {
		log.Fatalf("Error remove kafka config file, err:%v", err)
	}

	if err := messages.InitKfkLib(kafkaCfg, log); err != nil {
		log.Errorf("Error connecting kfk mq, err:%v", err)

		return
	}

	defer messages.KfkLibExit()

	// gitlab
	gitlab, err := platformimpl.NewPlatform(&cfg.Gitlab)
	if err != nil {
		log.Errorf("init gitlab platform failed, err:%s", err.Error())

		return
	}

	// obs service
	obsService, err := obsimpl.NewOBS(&cfg.OBS)
	if err != nil {
		log.Errorf("init obs service failed, err:%s", err.Error())

		return
	}

	// mysql
	if err := mysql.Init(&cfg.Mysql); err != nil {
		log.Errorf("init mysql failed, err:%s", err.Error())

		return
	}

	lock := synclockimpl.NewRepoSyncLock(mysql.NewSyncLockMapper())

	// sync service
	service := app.NewSyncService(&cfg.App, log, obsService, gitlab, lock)

	d := syncrepo.NewSyncRepo(&cfg.SyncRepo, service)
	if err != nil {
		log.Errorf("Error new dispatcherj, err:%s", err.Error())

		return
	}

	// run
	run(d, log)
}

func run(d *syncrepo.SyncRepo, log *logrus.Entry) {
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)

	var wg sync.WaitGroup
	defer wg.Wait()

	called := false
	ctx, done := context.WithCancel(context.Background())

	defer func() {
		if !called {
			called = true
			done()
		}
	}()

	wg.Add(1)
	go func(ctx context.Context) {
		defer wg.Done()

		select {
		case <-ctx.Done():
			log.Info("receive done. exit normally")
			return

		case <-sig:
			log.Info("receive exit signal")
			done()
			called = true
			return
		}
	}(ctx)

	if err := d.Run(ctx, log); err != nil {
		log.Errorf("subscribe failed, err:%v", err)
	}
}
