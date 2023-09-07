package messages

import (
	"errors"
	"io/ioutil"

	kfklib "github.com/opensourceways/kafka-lib/agent"
	kfkmq "github.com/opensourceways/kafka-lib/mq"
	redislib "github.com/opensourceways/redis-lib"
)

const (
	kfkQueueName      = "xihe-kafka-queue"
	kfkDefaultVersion = "2.3.0"
)

func InitKfkLib(kfkCfg kfklib.Config, log kfkmq.Logger) (err error) {
	return kfklib.Init(&kfkCfg, log, redislib.DAO(), kfkQueueName)
}

func KfkLibExit() {
	kfklib.Exit()
}

func LoadKafkaConfig(file string) (cfg kfklib.Config, err error) {
	v, err := ioutil.ReadFile(file)
	if err != nil {
		return
	}

	str := string(v)
	if str == "" {
		err = errors.New("missing addresses")

		return
	}

	cfg.Address = str
	cfg.Version = kfkDefaultVersion

	return
}
