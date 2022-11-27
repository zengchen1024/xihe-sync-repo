package domain

import "errors"

const (
	repoSyncStatusDone    = "done"
	repoSyncStatusRunning = "running"
)

var (
	RepoSyncStatusDone    = repoSyncStatus(repoSyncStatusDone)
	RepoSyncStatusRunning = repoSyncStatus(repoSyncStatusRunning)
)

// RepoSyncStatus
type RepoSyncStatus interface {
	RepoSyncStatus() string
	IsDone() bool
}

func NewRepoSyncStatus(s string) (RepoSyncStatus, error) {
	if s == "" {
		return nil, nil
	}

	if s != repoSyncStatusDone && s != repoSyncStatusRunning {
		return nil, errors.New("invalid repo sync status")
	}

	return repoSyncStatus(s), nil
}

type repoSyncStatus string

func (s repoSyncStatus) RepoSyncStatus() string {
	return string(s)
}

func (s repoSyncStatus) IsDone() bool {
	return string(s) == repoSyncStatusDone
}

type RepoSyncLock struct {
	Id         string
	Owner      Account
	RepoId     string
	Status     RepoSyncStatus
	Version    int
	LastCommit string
}
