package obs

type OBS interface {
	SaveObject(path, content string) error
	GetObject(path string) ([]byte, error)
	CopyObject(dst, src string) error
	OBSUtilPath() string
	OBSBucket() string
}
