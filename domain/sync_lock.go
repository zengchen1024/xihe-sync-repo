package domain

import (
	"errors"
	"time"
)

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

func NewRepoSyncLock(owner Account, repoId string) RepoSyncLock {
	return RepoSyncLock{
		Owner:  owner,
		RepoId: repoId,
	}
}

type RepoSyncLock struct {
	Id         string
	Owner      Account
	RepoId     string
	Status     RepoSyncStatus
	Expiry     int64
	Version    int
	LastCommit string
}

func (l *RepoSyncLock) Lock(commit string) bool {
	if l.LastCommit == commit {
		return false
	}

	l.Status = RepoSyncStatusRunning
	l.Expiry = time.Now().Unix() + 10*3600

	return true
}

func (l *RepoSyncLock) UnLock(commit string) {
	if commit != "" {
		l.LastCommit = commit
	}

	l.Status = RepoSyncStatusDone
	l.Expiry = 0
}

func (l *RepoSyncLock) IsDoing() bool {
	return !l.isDone() && !l.isExpiried()
}

func (l *RepoSyncLock) isDone() bool {
	return l.Status == nil || l.Status.IsDone()
}

func (l *RepoSyncLock) isExpiried() bool {
	return l.Expiry < time.Now().Unix()
}
