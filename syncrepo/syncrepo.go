package syncrepo

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"sync"

	kfklib "github.com/opensourceways/kafka-lib/agent"
	"github.com/sirupsen/logrus"

	"github.com/opensourceways/xihe-sync-repo/app"
)

const (
	retryNum         = 3
	headerTag        = "SEND-BACK-NUM"
	kfkConsumerGroup = "xihe-sync-repo"
)

type message struct {
	task   syncRepoTask
	body   []byte
	header map[string]string
}

type SyncRepo struct {
	topic       string
	generator   syncRepoTaskGenerator
	syncservice app.SyncService

	wg              sync.WaitGroup
	messageChan     chan message
	messageChanSize int
}

func NewSyncRepo(cfg *Config, service app.SyncService) *SyncRepo {
	size := cfg.concurrentSize()

	return &SyncRepo{
		topic: cfg.Topic,

		generator: syncRepoTaskGenerator{
			userAgent: cfg.UserAgent,
		},
		syncservice: service,

		messageChan:     make(chan message, size),
		messageChanSize: size,
	}
}

func (d *SyncRepo) Run(ctx context.Context, cfg *Config, log *logrus.Entry) error {
	if err := kfklib.Init(&cfg.Kafka, log, nil, "", true); err != nil {
		return err
	}

	defer kfklib.Exit()

	err := kfklib.SubscribeWithStrategyOfRetry(
		kfkConsumerGroup, d.handle, []string{d.topic}, retryNum,
	)
	if err != nil {
		return err
	}

	for i := 0; i < d.messageChanSize; i++ {
		d.wg.Add(1)

		go func() {
			d.doTask(log)
			d.wg.Done()
		}()
	}

	<-ctx.Done()

	close(d.messageChan)

	d.wg.Wait()

	return nil
}

func (d *SyncRepo) handle(body []byte, header map[string]string) error {
	if err := d.validateMessage(body, header); err != nil {
		// no need retry
		return nil
	}

	task, retry, err := d.generator.genTask(body, header)
	if err != nil {
		if retry {
			return err
		}

		return nil
	}

	d.messageChan <- message{
		task:   task,
		body:   body,
		header: header,
	}

	return nil
}

func (d *SyncRepo) validateMessage(body []byte, header map[string]string) error {
	if len(body) == 0 {
		return errors.New("unexpect message: empty payload")
	}

	if len(header) == 0 {
		return errors.New("unexpect message: empty header")
	}

	return nil
}

func (d *SyncRepo) doTask(log *logrus.Entry) {
	f := func(msg message) {
		task := &msg.task

		err := d.syncservice.SyncRepo(task)
		if err == nil {
			return
		}

		s := fmt.Sprintf(
			"%s/%s/%s", task.Owner.Account(), task.RepoName, task.RepoId,
		)

		log.Errorf("sync repo(%s) failed, err:%s", s, err.Error())

		// pubish again
		h := msg.header

		n := 0
		if v, ok := h[headerTag]; ok {
			n, _ = strconv.Atoi(v)
		}
		h[headerTag] = strconv.Itoa(n + 1)

		if err = kfklib.Publish(d.topic, h, msg.body); err != nil {
			log.Errorf(
				"send back the message for repo(%s) failed, err:%s",
				s, err.Error(),
			)
		}
	}

	for {
		msg, ok := <-d.messageChan
		if !ok {
			return
		}

		f(msg)
	}
}
