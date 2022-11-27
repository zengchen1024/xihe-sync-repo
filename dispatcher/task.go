package dispatcher

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/xanzy/go-gitlab"
)

type syncRepoTask struct {
	Owner    string
	RepoId   string
	RepoName string
}

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
	cmd.Owner = v[0]
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
