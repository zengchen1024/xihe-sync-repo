package syncrepo

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/xanzy/go-gitlab"

	"github.com/opensourceways/xihe-sync-repo/app"
	"github.com/opensourceways/xihe-sync-repo/domain"
)

type syncRepoTask = app.RepoInfo

type syncRepoTaskGenerator struct {
	userAgent string
}

func (d *syncRepoTaskGenerator) genTask(payload []byte, header map[string]string) (
	cmd syncRepoTask, ok bool, err error,
) {
	eventType, err := d.parseRequest(payload, header)
	if err != nil {
		err = fmt.Errorf("invalid task, err:%s", err.Error())

		return
	}

	if gitlab.EventType(eventType) != gitlab.EventTypePush {
		return
	}

	e := new(gitlab.PushEvent)
	if err = json.Unmarshal(payload, e); err != nil {
		return
	}

	v := strings.Split(e.Project.PathWithNamespace, "/")

	if cmd.Owner, err = domain.NewAccount(v[0]); err != nil {
		return
	}
	cmd.RepoName = v[1]
	cmd.RepoId = strconv.Itoa(e.ProjectID)

	ok = true

	return
}

func (d *syncRepoTaskGenerator) parseRequest(payload []byte, header map[string]string) (
	eventType string, err error,
) {
	if header == nil {
		err = errors.New("no header")

		return
	}

	if header["User-Agent"] != d.userAgent {
		err = errors.New("unknown User-Agent Header")

		return
	}

	if eventType = header["X-Gitlab-Event"]; eventType == "" {
		err = errors.New("missing X-Gitlab-Event Header")

		return
	}

	if header["X-Gitlab-Event-UUID"] == "" {
		err = errors.New("missing X-Gitlab-Event-UUID Header")
	}

	return
}
