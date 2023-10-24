package mysql

const (
	fieldStatus     = "status"
	fieldExpiry     = "expiry"
	fieldVersion    = "version"
	fieldLastCommit = "last_commit"
)

var tableName = ""

type RepoSyncLock struct {
	Id         int    `gorm:"column:id"`
	Owner      string `gorm:"column:owner"`
	RepoId     string `gorm:"column:repo_id"`
	Status     string `gorm:"column:status"`
	Expiry     int64  `gorm:"column:expiry"`
	Version    int    `gorm:"column:version"`
	LastCommit string `gorm:"column:last_commit"`
}

func (r *RepoSyncLock) TableName() string {
	return tableName
}
