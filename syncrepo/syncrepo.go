package syncrepo

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"sync"

	"github.com/opensourceways/community-robot-lib/utils"
	kfklib "github.com/opensourceways/kafka-lib/agent"
	"github.com/sirupsen/logrus"

	"github.com/opensourceways/xihe-sync-repo/app"
	"github.com/opensourceways/xihe-sync-repo/infrastructure/messages"
)

const (
	retryNum           = 3
	handlerNameGenTask = "create_task"
)

type message struct {
	msg  messages.Message
	task syncRepoTask
}

type SyncRepo struct {
	hmac        string
	topic       string
	endpoint    string
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
		hmac:     cfg.AccessHmac,
		topic:    cfg.Topic,
		endpoint: cfg.AccessEndpoint,

		hc: utils.NewHttpClient(3),
		generator: syncRepoTaskGenerator{
			userAgent: cfg.UserAgent,
		},
		syncservice: service,

		messageChan:     make(chan message, size),
		messageChanSize: size,
	}
}

func (d *SyncRepo) Run(ctx context.Context, log *logrus.Entry) error {
	if err := kfklib.SubscribeWithStrategyOfRetry(
		handlerNameGenTask, d.handle, []string{d.topic}, retryNum,
	); err != nil {
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
		return err
	}

	task, ok, err := d.generator.genTask(body, header)
	if err != nil || !ok {
		return err
	}

	msg := messages.Message{
		Body:   body,
		Header: header,
	}

	d.messageChan <- message{
		msg:  msg,
		task: task,
	}

	return nil
}

func (d *SyncRepo) validateMessage(body []byte, header map[string]string) error {
	if len(body) == 0 {
		return errors.New("unexpect message: empty header")
	}

	if len(header) == 0 {
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

		if err = d.sendBack(msg.msg.Body); err != nil {
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

func (d *SyncRepo) sendBack(body []byte) error {
	req, err := http.NewRequest(
		http.MethodPost, d.endpoint, bytes.NewBuffer(body),
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
