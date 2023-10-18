package mysql

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"github.com/go-sql-driver/mysql"
	"io/ioutil"
	"os"
	"time"

	gormmysql "gorm.io/driver/mysql"
	"gorm.io/gorm"
)

var cli *mysqlService

func Init(cfg *Config) error {
	ca, err := ioutil.ReadFile(cfg.DBCert)
	if err != nil {
		return err
	}

	if err := os.Remove(cfg.DBCert); err != nil {
		return err
	}

	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(ca) {
		return fmt.Errorf("faild to append certs from PEM")
	}

	tlsConfig := &tls.Config{
		RootCAs:            pool,
		InsecureSkipVerify: true,
	}

	mysqlConfig := mysql.Config{
		TLS: tlsConfig,
	}

	config := gormmysql.Config{
		DSN:                       cfg.Conn,
		DontSupportRenameIndex:    true,
		DontSupportRenameColumn:   true,
		SkipInitializeWithVersion: false,
		DSNConfig:                 &mysqlConfig,
	}

	db, err := gorm.Open(gormmysql.New(config), &gorm.Config{})
	if err != nil {
		return err
	}

	sqlDB, err := db.DB()
	if err != nil {
		return err
	}

	sqlDB.SetConnMaxLifetime(time.Duration(cfg.ConnMaxLifetime) * time.Second)
	sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
	sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)

	cli = &mysqlService{
		db: db,
	}

	tableName = cfg.TableName

	return nil
}

type mysqlService struct {
	db *gorm.DB
}
