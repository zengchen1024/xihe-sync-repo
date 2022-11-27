package mysql

import (
	"errors"
	"strconv"

	"gorm.io/gorm"

	"github.com/opensourceways/xihe-sync-repo/infrastructure/synclockimpl"
)

func NewSyncLockMapper() synclockimpl.SyncLockMapper {
	return syncLock{}
}

type syncLock struct{}

func (rs syncLock) Insert(do *synclockimpl.RepoSyncLockDO) (string, error) {
	table := rs.toSyncLockTable(do)

	r := cli.db.Model(&table).Create(&table)
	if r.Error != nil {
		return "", r.Error
	}

	if r.RowsAffected == 0 {
		return "", synclockimpl.NewErrorDuplicateCreating(
			errors.New("duplecate creating"),
		)
	}

	return strconv.Itoa(table.Id), nil
}

func (rs syncLock) Get(owner, repoId string) (do synclockimpl.RepoSyncLockDO, err error) {
	cond := &RepoSyncLock{
		Owner:  owner,
		RepoId: repoId,
	}

	data := new(RepoSyncLock)

	err = cli.db.Model(data).Where(cond).First(data).Error

	if err == nil {
		do = rs.toSyncLockDo(data)
	} else {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			err = synclockimpl.NewErrorDataNotExists(err)
		}
	}

	return
}

func (rs syncLock) Update(do *synclockimpl.RepoSyncLockDO) error {
	cond := &RepoSyncLock{
		Owner:   do.Owner,
		RepoId:  do.RepoId,
		Version: do.Version,
	}

	tx := cli.db.Model(cond).Where(cond).Updates(
		map[string]interface{}{
			fieldVersion:    gorm.Expr(fieldVersion+" + ?", 1),
			fieldLastCommit: do.LastCommit,
			fieldStatus:     do.Status,
		},
	)
	if tx.Error != nil {
		return tx.Error
	}

	if tx.RowsAffected == 0 {
		return synclockimpl.NewErrorConcurrentUpdating(
			errors.New("no matched record"),
		)
	}

	return nil
}

func (rs syncLock) toSyncLockTable(do *synclockimpl.RepoSyncLockDO) RepoSyncLock {
	return RepoSyncLock{
		Owner:      do.Owner,
		RepoId:     do.RepoId,
		Status:     do.Status,
		Version:    do.Version,
		LastCommit: do.LastCommit,
	}
}

func (rs syncLock) toSyncLockDo(data *RepoSyncLock) synclockimpl.RepoSyncLockDO {
	return synclockimpl.RepoSyncLockDO{
		Id:         strconv.Itoa(data.Id),
		Owner:      data.Owner,
		RepoId:     data.RepoId,
		Status:     data.Status,
		Version:    data.Version,
		LastCommit: data.LastCommit,
	}
}
