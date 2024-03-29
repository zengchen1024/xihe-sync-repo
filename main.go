package main

import (
	"context"
	"errors"
	"flag"
	"io/ioutil"
	"os"
	"os/signal"
	"regexp"
	"strings"
	"sync"
	"syscall"

	"github.com/opensourceways/community-robot-lib/kafka"
	"github.com/opensourceways/community-robot-lib/logrusutil"
	"github.com/opensourceways/community-robot-lib/mq"
	liboptions "github.com/opensourceways/community-robot-lib/options"
	"github.com/sirupsen/logrus"

	"github.com/opensourceways/xihe-sync-repo/app"
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

	// init kafka
	kafkaCfg, err := loadKafkaConfig(o.kafkamqConfigFile)
	if err != nil {
		log.Errorf("Error loading kfk config, err:%v", err)

		return
	}

	if err := connetKafka(&kafkaCfg); err != nil {
		log.Errorf("Error connecting kfk mq, err:%v", err)

		return
	}

	defer kafka.Disconnect()

	// load config
	cfg, err := loadConfig(o.service.ConfigFile)
	if err != nil {
		log.Errorf("Error loading config, err:%v", err)

		return
	}

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

func connetKafka(cfg *mq.MQConfig) error {
	tlsConfig, err := cfg.TLSConfig.TLSConfig()
	if err != nil {
		return err
	}

	err = kafka.Init(
		mq.Addresses(cfg.Addresses...),
		mq.SetTLSConfig(tlsConfig),
		mq.Log(logrus.WithField("module", "kfk")),
	)
	if err != nil {
		return err
	}

	return kafka.Connect()
}

func loadKafkaConfig(file string) (cfg mq.MQConfig, err error) {
	v, err := ioutil.ReadFile(file)
	if err != nil {
		return
	}

	str := string(v)
	if str == "" {
		err = errors.New("missing addresses")

		return
	}

	addresses := parseAddress(str)
	if len(addresses) == 0 {
		err = errors.New("no valid address for kafka")

		return
	}

	if err = kafka.ValidateConnectingAddress(addresses); err != nil {
		return
	}

	cfg.Addresses = addresses

	return
}

func parseAddress(addresses string) []string {
	var reIpPort = regexp.MustCompile(`^((25[0-5]|(2[0-4]|1\d|[1-9]|)\d)\.?\b){4}:[1-9][0-9]*$`)

	v := strings.Split(addresses, ",")
	r := make([]string, 0, len(v))
	for i := range v {
		if reIpPort.MatchString(v[i]) {
			r = append(r, v[i])
		}
	}

	return r
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
