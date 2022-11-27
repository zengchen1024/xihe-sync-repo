package syncrepo

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"sync"

	"github.com/opensourceways/community-robot-lib/kafka"
	"github.com/opensourceways/community-robot-lib/mq"
	"github.com/opensourceways/community-robot-lib/utils"
	"github.com/sirupsen/logrus"

	"github.com/opensourceways/xihe-sync-repo/app"
)

type message struct {
	msg  *mq.Message
	task syncRepoTask
}

type SyncRepo struct {
	endpoint    string
	hmac        string
	topic       string
	hc          utils.HttpClient
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
		hc:    utils.NewHttpClient(3),
		generator: syncRepoTaskGenerator{
			userAgent: cfg.UserAgent,
		},
		syncservice: service,

		messageChan:     make(chan message, size),
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

	d.messageChan <- message{
		msg:  msg,
		task: task,
	}

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
	f := func(msg message) (err error) {
		task := &msg.task
		if err = d.syncservice.SyncRepo(task); err == nil {
			return nil
		}

		s := fmt.Sprintf(
			"%s/%s/%s", task.Owner.Account(), task.RepoName, task.RepoId,
		)
		log.Errorf("sync repo(%s) failed, err:%s", s, err.Error())

		if err = d.sendBack(msg.msg); err != nil {
			log.Errorf(
				"send back the message for repo(%s) failed, err:%s",
				s, err.Error(),
			)
		}

		return nil
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

func (d *SyncRepo) sendBack(e *mq.Message) error {
	req, err := http.NewRequest(
		http.MethodPost, d.endpoint, bytes.NewBuffer(e.Body),
	)
	if err != nil {
		return err
	}

	h := &req.Header
	h.Add("Content-Type", "application/json")
	h.Add("User-Agent", "xihe-sync-repo")
	h.Add("X-Gitlab-Event", "System Hook")
	h.Add("X-Gitlab-Token", d.hmac)
	h.Add("X-Gitlab-Event-UUID", "73ed8438-1119-4bb8-ae9d-0180c88ef168")

	_, err = d.hc.ForwardTo(req, nil)

	return err
}
