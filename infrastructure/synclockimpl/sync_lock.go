package synclockimpl

import (
	"github.com/opensourceways/xihe-sync-repo/domain"
	"github.com/opensourceways/xihe-sync-repo/domain/synclock"
)

type SyncLockMapper interface {
	Insert(*RepoSyncLockDO) (string, error)
	Update(*RepoSyncLockDO) error
	Get(string, string) (RepoSyncLockDO, error)
}

func NewRepoSyncLock(mapper SyncLockMapper) synclock.RepoSyncLock {
	return syncLock{mapper}
}

type syncLock struct {
	mapper SyncLockMapper
}

func (impl syncLock) Save(p *domain.RepoSyncLock) (r domain.RepoSyncLock, err error) {
	do := impl.toRepoSyncLockDO(p)

	if p.Id != "" {
		if err = impl.mapper.Update(&do); err != nil {
			err = convertError(err)
		} else {
			r = *p
			r.Version += 1
		}

		return
	}

	v, err := impl.mapper.Insert(&do)
	if err != nil {
		err = convertError(err)
	} else {
		r = *p
		r.Id = v
	}

	return
}

func (impl syncLock) Find(owner domain.Account, repoId string) (
	r domain.RepoSyncLock, err error,
) {
	v, err := impl.mapper.Get(owner.Account(), repoId)
	if err != nil {
		err = convertError(err)
	} else {
		err = v.toSyncLock(&r)
	}

	return
}

func (impl syncLock) toRepoSyncLockDO(p *domain.RepoSyncLock) RepoSyncLockDO {
	return RepoSyncLockDO{
		Id:         p.Id,
		Owner:      p.Owner.Account(),
		RepoId:     p.RepoId,
		Status:     p.Status.RepoSyncStatus(),
		Expiry:     p.Expiry,
		Version:    p.Version,
		LastCommit: p.LastCommit,
	}
}

type RepoSyncLockDO struct {
	Id         string
	Owner      string
	RepoId     string
	Status     string
	Expiry     int64
	Version    int
	LastCommit string
}

func (do *RepoSyncLockDO) toSyncLock(r *domain.RepoSyncLock) (err error) {
	if r.Owner, err = domain.NewAccount(do.Owner); err != nil {
		return
	}

	if r.Status, err = domain.NewRepoSyncStatus(do.Status); err != nil {
		return
	}

	r.Id = do.Id
	r.RepoId = do.RepoId
	r.Expiry = do.Expiry
	r.Version = do.Version
	r.LastCommit = do.LastCommit

	return
}
