package app

import (
	"path/filepath"

	"github.com/opensourceways/xihe-sync-repo/domain/obs"
	"github.com/opensourceways/xihe-sync-repo/utils"
)

type syncHelper struct {
	obsService obs.OBS
	cfg        HelperConfig
}

// sha: sha
// dst: user/[project,model,dataset]/repo_id/xxx
func (s *syncHelper) syncLFSFile(sha, dst string) error {
	return utils.Retry(func() error {
		return s.obsService.CopyObject(
			filepath.Join(s.cfg.RepoPath, dst),
			filepath.Join(s.cfg.LFSPath, sha[:2], sha[2:4], sha[4:]),
		)
	})
}

// p: user/[project,model,dataset]/repo_id
func (s *syncHelper) saveLastCommit(p, commit string) error {
	return utils.Retry(func() error {
		return s.obsService.SaveObject(
			filepath.Join(s.cfg.RepoPath, p, s.cfg.CommitFile),
			commit,
		)
	})
}

// p: user/[project,model,dataset]/repo_id
func (s *syncHelper) getRepoObsPath(p string) string {
	return filepath.Join(s.cfg.RepoPath, p)
}
