package syncrepo

import (
	"context"
	"errors"
	"sync"

	"github.com/opensourceways/community-robot-lib/kafka"
	"github.com/opensourceways/community-robot-lib/mq"
	"github.com/sirupsen/logrus"

	"github.com/opensourceways/xihe-sync-repo/app"
	"github.com/opensourceways/xihe-sync-repo/domain"
)

type SyncRepo struct {
	topic       string
	generator   syncRepoTaskGenerator
	syncservice app.SyncService

	wg              sync.WaitGroup
	messageChan     chan syncRepoTask
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

		messageChan:     make(chan syncRepoTask, size),
		messageChanSize: size,
	}
}

func (d *SyncRepo) Run(ctx context.Context, log *logrus.Entry) error {
	s, err := kafka.Subscribe(
		d.topic,
		d.handle,
		func(opt *mq.SubscribeOptions) {
			opt.Queue = "xihe-sync-repo"
		},
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

	s.Unsubscribe()

	close(d.messageChan)

	d.wg.Wait()

	return nil
}

func (d *SyncRepo) handle(event mq.Event) error {
	msg := event.Message()

	if err := d.validateMessage(msg); err != nil {
		return err
	}

	task, ok, err := d.generator.genTask(msg.Body, msg.Header)
	if err != nil || !ok {
		return err
	}

	d.messageChan <- task

	return nil
}

func (d *SyncRepo) validateMessage(msg *mq.Message) error {
	if msg == nil {
		return errors.New("get a nil msg from broker")
	}

	if len(msg.Header) == 0 {
		return errors.New("unexpect message: empty header")
	}

	if len(msg.Body) == 0 {
		return errors.New("unexpect message: empty payload")
	}

	return nil
}

func (d *SyncRepo) doTask(log *logrus.Entry) {
	f := func(task syncRepoTask) error {
		owner, err := domain.NewAccount(task.Owner)
		if err != nil {
			return err
		}

		return d.syncservice.SyncRepo(&app.RepoInfo{
			Owner:    owner,
			RepoId:   task.RepoId,
			RepoName: task.RepoName,
		})
	}

	for {
		msg, ok := <-d.messageChan
		if !ok {
			return
		}

		if err := f(msg); err != nil {
			log.Errorf("do task failed, err:%s", err.Error())
		}
	}
}
