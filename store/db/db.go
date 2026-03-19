package db

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
)

type DBIf interface {
	Raw() *sqlx.DB // 获取原生的 sqlx.DB 对象

	PingContext(ctx context.Context) error
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)

	// 预处理
	PrepareContext(ctx context.Context, query string) (*sql.Stmt, error)

	// 事务相关
	BeginTxx(ctx context.Context, opts *sql.TxOptions) (*sqlx.Tx, error)
}

type DB struct {
	*sqlx.DB
}

func (db *DB) Raw() *sqlx.DB {
	return db.DB
}

func Open(config *Config) (*DB, error) {
	if config == nil {
		return nil, fmt.Errorf("mysql config is nil")
	}
	// 创建mysql配置
	mysqlConfig := mysql.NewConfig()
	mysqlConfig.User = config.UserName
	mysqlConfig.Passwd = config.Password
	mysqlConfig.Net = "tcp"
	mysqlConfig.Addr = fmt.Sprintf("%s:%d", config.Host, config.Port)
	mysqlConfig.DBName = config.DataBase
	mysqlConfig.Loc = time.Local
	mysqlConfig.Timeout = config.DialTimeout
	mysqlConfig.ReadTimeout = config.ReadTimeout
	mysqlConfig.WriteTimeout = config.WriteTimeout

	// 创建数据库连接
	dsn := mysqlConfig.FormatDSN()
	db, err := sqlx.Connect("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("mysql connect failed: %w", err)
	}

	err = db.Ping()
	if err != nil {
		return nil, fmt.Errorf("mysql ping failed: %w", err)
	}

	// 最大连接数
	db.SetMaxOpenConns(config.MaxOpenConns)
	// 最大空闲连接数
	db.SetMaxIdleConns(config.MaxIdleConns)
	// 最大连接生命周期
	db.SetConnMaxLifetime(time.Duration(config.ConnMaxLifetime) * time.Second)

	return &DB{db}, nil
}

func Close(db *DB) {
	db.DB.Close()
}
