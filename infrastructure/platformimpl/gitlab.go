package platformimpl

import (
	"fmt"
	"strings"

	gitlab "github.com/xanzy/go-gitlab"

	"github.com/opensourceways/xihe-sync-repo/domain/platform"
)

func NewPlatform(cfg *Config) (platform.Platform, error) {
	cli, err := gitlab.NewOAuthClient(
		cfg.Token,
		gitlab.WithBaseURL(cfg.Host),
	)
	if err != nil {
		return nil, err
	}

	u, _, err := cli.Users.CurrentUser()
	if err != nil {
		return nil, err
	}

	return &platformImpl{
		cli: cli,
		endpoint: strings.Replace(
			strings.TrimSuffix(cfg.Host, "/"), "://",
			fmt.Sprintf("://%s:%s@", u.Username, cfg.Token), 1,
		),
	}, nil
}

type platformImpl struct {
	cli      *gitlab.Client
	endpoint string
}

func (h *platformImpl) GetCloneURL(owner, repo string) string {
	return fmt.Sprintf("%s/%s/%s", h.endpoint, owner, repo)
}

func (h *platformImpl) GetLastCommit(pid string) (string, error) {
	opts := gitlab.ListCommitsOptions{}
	opts.Page = 1
	opts.PerPage = 1

	v, resp, err := h.cli.Commits.ListCommits(pid, &opts, nil)
	if err != nil {
		if resp.StatusCode == 404 {
			err = platform.NewErrorRepoNotExists(err)
		}

		return "", err
	}

	if len(v) == 0 {
		return "", nil
	}

	return v[0].ID, nil
}
