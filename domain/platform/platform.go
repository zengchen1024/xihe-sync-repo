package platform

type Platform interface {
	GetLastCommit(pid string) (string, error)
	GetCloneURL(owner, repo string) string
}
