package app

import (
	"errors"
	"fmt"
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

		c = domain.NewRepoSyncLock(info.Owner, info.RepoId)
	}

	if c.IsDoing() {
		return errors.New("can't sync")
	}

	lastCommit, err := s.ph.GetLastCommit(info.RepoId)
	if err != nil {
		if platform.IsErrorRepoNotExists(err) {
			return nil
		}

		return err
	}

	// try lock
	if b, err := s.tryLock(&c, lastCommit); !b || err != nil {
		return err
	}

	// do sync
	lastCommit, syncErr := s.doSync(c.LastCommit, info)

	// unlock
	s.tryUnlock(&c, lastCommit)

	return syncErr
}

func (s *syncService) tryLock(c *domain.RepoSyncLock, commit string) (bool, error) {
	if !c.Lock(commit, s.cfg.Expiry) {
		// the repo is up to date now.
		return false, nil
	}

	c1, err := s.lock.Save(c)
	if err == nil {
		*c = c1
	}

	return true, err
}

func (s *syncService) tryUnlock(c *domain.RepoSyncLock, commit string) {
	c.UnLock(commit)

	err := utils.Retry(func() error {
		_, err1 := s.lock.Save(c)
		if err1 != nil {
			s.log.Errorf("save lock(%v) failed, err:%s", *c, err1.Error())
		}

		return err1
	})

	if err != nil {
		s.log.Errorf("save lock(%s) failed, dead lock happened", *c)
	}
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
	tempDir, err := os.MkdirTemp(s.cfg.WorkDir, "sync")
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

	v, err, _ := utils.RunCmd(nil, params...)
	if err != nil {
		params[2] = "clone_url"
		err = fmt.Errorf(
			"run sync shell, err=%s, params=%v,",
			err.Error(), params,
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
