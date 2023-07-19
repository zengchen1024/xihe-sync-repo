package app

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/sirupsen/logrus"

	"github.com/opensourceways/xihe-sync-repo/domain"
	"github.com/opensourceways/xihe-sync-repo/domain/obs"
	"github.com/opensourceways/xihe-sync-repo/domain/platform"
	"github.com/opensourceways/xihe-sync-repo/domain/synclock"
	"github.com/opensourceways/xihe-sync-repo/utils"
)

type RepoInfo struct {
	Owner    domain.Account
	RepoId   string
	RepoName string
}

func (s *RepoInfo) repoOBSPath() string {
	return filepath.Join(s.Owner.Account(), s.RepoId)
}

type SyncService interface {
	SyncRepo(*RepoInfo) error
}

func NewSyncService(
	cfg *Config, log *logrus.Entry,
	s obs.OBS,
	p platform.Platform,
	l synclock.RepoSyncLock,
) SyncService {
	return &syncService{
		h: &syncHelper{
			obsService: s,
			cfg:        cfg.HelperConfig,
		},
		log:       log,
		cfg:       cfg.ServiceConfig,
		lock:      l,
		ph:        p,
		obsutil:   s.OBSUtilPath(),
		obsBucket: s.OBSBucket(),
	}
}

type syncService struct {
	h   *syncHelper
	log *logrus.Entry
	cfg ServiceConfig

	obsutil   string
	obsBucket string

	lock synclock.RepoSyncLock
	ph   platform.Platform
}

func (s *syncService) SyncRepo(info *RepoInfo) error {
	c, err := s.lock.Find(info.Owner, info.RepoId)
	if err != nil {
		if !synclock.IsRepoSyncLockNotExist(err) {
			return err
		}

		c.Owner = info.Owner
		c.RepoId = info.RepoId
	}

	if c.Status != nil && !c.Status.IsDone() {
		// TODO: mybe dead lock, try to unlock it and continue

		return errors.New("can't sync")
	}

	lastCommit, err := s.ph.GetLastCommit(info.RepoId)
	if err != nil {
		if platform.IsErrorRepoNotExists(err) {
			return nil
		}

		return err
	}
	if c.LastCommit == lastCommit {
		return nil
	}

	// try lock
	c.Status = domain.RepoSyncStatusRunning
	c, err = s.lock.Save(&c)
	if err != nil {
		return err
	}

	// do sync
	lastCommit, syncErr := s.doSync(c.LastCommit, info)
	if syncErr == nil {
		c.LastCommit = lastCommit
	}
	c.Status = domain.RepoSyncStatusDone

	// unlock
	err = utils.Retry(func() error {
		_, err := s.lock.Save(&c)
		if err != nil {
			s.log.Errorf(
				"save sync repo(%s) failed, err:%s, value=%v",
				info.repoOBSPath(), err.Error(), c,
			)
		}

		return err
	})
	if err != nil {
		s.log.Errorf(
			"save sync repo(%s) failed, dead lock happened",
			info.repoOBSPath(),
		)
	}

	return syncErr
}

func (s *syncService) doSync(startCommit string, info *RepoInfo) (lastCommit string, err error) {
	if lastCommit, err = s.sync(startCommit, info); err != nil {
		return
	}

	err = s.h.saveLastCommit(info.repoOBSPath(), lastCommit)
	if err != nil {
		s.log.Errorf(
			"update last commit failed, err:%s",
			err.Error(),
		)

		err = errors.New(
			"sync successfully , but save last commit to obs failed",
		)
	}

	return
}

func (s *syncService) sync(startCommit string, info *RepoInfo) (last string, err error) {
	tempDir, err := ioutil.TempDir(s.cfg.WorkDir, "sync")
	if err != nil {
		return
	}

	defer os.RemoveAll(tempDir)

	last, lfsFile, err := s.syncFile(tempDir, startCommit, info)
	s.log.Debugf(
		"sync file for repo:%s, last commit=%s, lfsFile=%s",
		info.repoOBSPath(), last, lfsFile,
	)
	if err != nil || lfsFile == "" {
		return
	}

	err = s.syncLFSFiles(lfsFile, info)

	return
}

func (s *syncService) syncLFSFiles(lfsFiles string, info *RepoInfo) error {
	obsPath := info.repoOBSPath()

	return utils.ReadFileLineByLine(lfsFiles, func(line string) error {
		v := strings.Split(line, ":oid sha256:")
		dst := filepath.Join(obsPath, v[0])

		s.log.Debugf("save lfs %s to %s", v[1], dst)

		return s.h.syncLFSFile(v[1], dst)
	})
}

func (s *syncService) syncFile(workDir, startCommit string, info *RepoInfo) (
	lastCommit string, lfsFile string, err error,
) {
	params := []string{
		s.cfg.SyncFileShell,
		workDir,
		s.ph.GetCloneURL(info.Owner.Account(), info.RepoName),
		info.RepoName, s.obsutil, s.obsBucket,
		s.h.getRepoObsPath(info.repoOBSPath()),
		startCommit,
	}

	v, err, _ := utils.RunCmd(params...)
	if err != nil {
		err = fmt.Errorf(
			"run sync shell, err=%s",
			err.Error(),
		)

		return
	}

	r := strings.Split(string(v), ", ")
	lastCommit = r[0]

	if strings.HasPrefix(r[2], "yes") {
		lfsFile = r[1]
	}

	return
}
