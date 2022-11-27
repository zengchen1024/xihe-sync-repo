package synclockimpl

import "github.com/opensourceways/xihe-sync-repo/domain/synclock"

type errorDuplicateCreating struct {
	error
}

func NewErrorDuplicateCreating(err error) errorDuplicateCreating {
	return errorDuplicateCreating{err}
}

type errorDataNotExists struct {
	error
}

func NewErrorDataNotExists(err error) errorDataNotExists {
	return errorDataNotExists{err}
}

type errorConcurrentUpdating struct {
	error
}

func NewErrorConcurrentUpdating(err error) errorConcurrentUpdating {
	return errorConcurrentUpdating{err}
}

func convertError(err error) (out error) {
	switch err.(type) {
	case errorDataNotExists:
		out = synclock.NewErrorRepoNotExists(err)

	default:
		out = err
	}

	return
}
