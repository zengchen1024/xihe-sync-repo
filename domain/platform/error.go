package platform

// errorRepoNotExists
type errorRepoNotExists struct {
	error
}

func NewErrorRepoNotExists(err error) errorRepoNotExists {
	return errorRepoNotExists{err}
}

// helper
func IsErrorRepoNotExists(err error) bool {
	_, ok := err.(errorRepoNotExists)

	return ok
}
