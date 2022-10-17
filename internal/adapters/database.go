package adapters

import (
	"os"
	"path/filepath"
	"time"

	"github.com/h44z/wg-portal/internal/config"

	"github.com/h44z/wg-portal/internal/model"
	"github.com/pkg/errors"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/driver/sqlserver"
	"gorm.io/gorm"
)

func NewDatabase(cfg config.DatabaseConfig) (*gorm.DB, error) {
	var gormDb *gorm.DB
	var err error

	switch cfg.Type {
	case config.DatabaseMySQL:
		gormDb, err = gorm.Open(mysql.Open(cfg.DSN), &gorm.Config{})
		if err != nil {
			return nil, errors.WithMessage(err, "failed to open MySQL database")
		}

		sqlDB, _ := gormDb.DB()
		sqlDB.SetConnMaxLifetime(time.Minute * 5)
		sqlDB.SetMaxIdleConns(2)
		sqlDB.SetMaxOpenConns(10)
		err = sqlDB.Ping() // This DOES open a connection if necessary. This makes sure the database is accessible
		if err != nil {
			return nil, errors.WithMessage(err, "failed to ping MySQL database")
		}
	case config.DatabaseMsSQL:
		gormDb, err = gorm.Open(sqlserver.Open(cfg.DSN), &gorm.Config{})
		if err != nil {
			return nil, errors.WithMessage(err, "failed to open sqlserver database")
		}
	case config.DatabasePostgres:
		gormDb, err = gorm.Open(postgres.Open(cfg.DSN), &gorm.Config{})
		if err != nil {
			return nil, errors.WithMessage(err, "failed to open Postgres database")
		}
	case config.DatabaseSQLite:
		if _, err = os.Stat(filepath.Dir(cfg.DSN)); os.IsNotExist(err) {
			if err = os.MkdirAll(filepath.Dir(cfg.DSN), 0700); err != nil {
				return nil, errors.WithMessage(err, "failed to create database base directory")
			}
		}
		gormDb, err = gorm.Open(sqlite.Open(cfg.DSN), &gorm.Config{DisableForeignKeyConstraintWhenMigrating: true})
		if err != nil {
			return nil, errors.WithMessage(err, "failed to open sqlite database")
		}
	}

	_ = gormDb.AutoMigrate(&model.Interface{}, &model.User{})
	_ = gormDb.AutoMigrate(&model.Peer{})

	return gormDb, nil
}
