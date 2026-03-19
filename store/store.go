package store

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/shuaibizhang/codecoverage/store/db"

	"github.com/didi/gendry/builder"
	"github.com/didi/gendry/scanner"
	"github.com/jmoiron/sqlx"
)

var TagName = "ddb"

type Store struct {
	db      db.DBIf
	logger  Logger
	metrics Metrics
}

type PrimaryKey struct {
	ID uint64 `json:"id" ddb:"id"`
}

var (
	ErrInvalidArguments = errors.New("invalid arguments")
	ErrRecordNotFound   = errors.New("record not found")
)

func NewStore(db db.DBIf) *Store {
	return &Store{
		db:      db,
		logger:  &NoopLogger{},
		metrics: &NoopMetrics{},
	}
}

// WithLogger 设置日志接口
func (s *Store) WithLogger(logger Logger) *Store {
	s.logger = logger
	return s
}

// WithMetrics 设置监控接口
func (s *Store) WithMetrics(metrics Metrics) *Store {
	s.metrics = metrics
	return s
}

// QueryByID 根据ID查询指定记录, val 为对应类型的指针
func (s *Store) QueryByID(ctx context.Context, table string, id uint64, vals interface{}) error {
	cond := NewCond().ID(id).Limit(0, 1)
	sql, values, err := builder.BuildSelect(table, map[string]interface{}(cond), nil)
	if err != nil {
		return err
	}

	rows, err := s.db.QueryContext(ctx, sql, values...)
	if err != nil {
		return err
	}

	err = scanner.ScanClose(rows, vals)
	if err != nil {
		return err
	}
	return nil
}

// Query 根据条件查询多条记录, vals 为对应类型的 slice 指针
func (s *Store) Query(ctx context.Context, table string, cond Cond, selectField []string, vals interface{}) error {
	start := time.Now()
	defer func() {
		duration := time.Since(start).Seconds()
		s.metrics.RecordDuration(ctx, "db_query_duration", duration, map[string]string{"table": table})
	}()

	sql, values, err := builder.BuildSelect(table, map[string]interface{}(cond), selectField)
	if err != nil {
		s.logger.Error(ctx, "build select sql failed", "table", table, "error", err)
		return err
	}

	rows, err := s.db.QueryContext(ctx, sql, values...)
	if err != nil {
		s.logger.Error(ctx, "query context failed", "table", table, "sql", sql, "error", err)
		return err
	}

	err = scanner.ScanClose(rows, vals)
	if err != nil {
		s.logger.Error(ctx, "scan rows failed", "table", table, "error", err)
		return err
	}
	return nil
}

// UpdateRecord 更新数据
func (s *Store) UpdateRecord(ctx context.Context, table string, cond Cond, id uint64, val interface{}) error {
	start := time.Now()
	defer func() {
		duration := time.Since(start).Seconds()
		s.metrics.RecordDuration(ctx, "db_update_duration", duration, map[string]string{"table": table})
	}()

	if val == nil {
		return ErrInvalidArguments
	}

	data, err := scanner.Map(val, TagName)
	if err != nil {
		return err
	}
	DeleteAutoColumns(data)

	var query string
	var values []interface{}
	if id == 0 {
		return errors.New("id can not be 0")
	}
	if cond == nil {
		cond = NewCond()
	}
	cond = cond.ID(id)
	query, values, err = builder.BuildUpdate(table, map[string]interface{}(cond), data)

	if err != nil {
		return err
	}

	result, err := s.db.ExecContext(ctx, query, values...)
	if err != nil {
		s.logger.Error(ctx, "update record failed", "table", table, "sql", query, "error", err)
		return err
	}

	_, err = result.RowsAffected()
	if err != nil {
		s.logger.Error(ctx, "get rows affected failed", "table", table, "error", err)
		return err
	}
	return nil
}

// Prepare 预处理封装
func (s *Store) Prepare(ctx context.Context, query string) (*sql.Stmt, error) {
	return s.db.PrepareContext(ctx, query)
}

// Tx 事务封装
func (s *Store) Tx(ctx context.Context, fn func(ctx context.Context, tx *sqlx.Tx) error) error {
	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}

	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p)
		} else if err != nil {
			tx.Rollback()
		} else {
			err = tx.Commit()
		}
	}()

	err = fn(ctx, tx)
	return err
}

// SaveRecord 更新或插入数据
func (s *Store) SaveRecord(ctx context.Context, table string, id uint64, val interface{}) (uint64, error) {
	if val == nil {
		return 0, ErrInvalidArguments
	}

	data, err := scanner.Map(val, TagName)
	if err != nil {
		return 0, err
	}
	DeleteAutoColumns(data)

	var query string
	var values []interface{}
	if id == 0 {
		rows := []map[string]interface{}{data}
		query, values, err = builder.BuildInsert(table, rows)
	} else {
		cond := NewCond().ID(id)
		query, values, err = builder.BuildUpdate(table, map[string]interface{}(cond), data)
	}
	if err != nil {
		return 0, err
	}

	result, err := s.db.ExecContext(ctx, query, values...)
	if err != nil {
		return 0, err
	}

	if id == 0 {
		insertID, err := result.LastInsertId()
		if err != nil {
			return 0, err
		}
		if insertID == 0 {
			return 0, errors.New("last insert id can not be 0")
		}
		id = uint64(insertID)
	} else {
		_, err = result.RowsAffected()
		if err != nil {
			return 0, err
		}
	}
	return id, nil
}

func (s *Store) BatchSaveRecord(ctx context.Context, table string, vals []interface{}) error {
	if len(vals) == 0 {
		return ErrInvalidArguments
	}
	var rows []map[string]interface{}
	for _, val := range vals {
		data, err := scanner.Map(val, TagName)
		if err != nil {
			return err
		}
		DeleteAutoColumns(data)
		rows = append(rows, data)
	}
	var query string
	var values []interface{}
	var err error
	query, values, err = builder.BuildInsert(table, rows)

	if err != nil {
		return err
	}

	_, err = s.db.ExecContext(ctx, query, values...)
	return err
}

// DeleteByID 删除指定ID的记录
func (s *Store) DeleteByID(ctx context.Context, table string, id uint64) error {
	update := map[string]interface{}{
		ColumnDeleted: 1,
	}
	cond := NewCond().ID(id)
	sql, values, err := builder.BuildUpdate(table, map[string]interface{}(cond), update)
	if err != nil {
		return err
	}
	result, err := s.db.ExecContext(ctx, sql, values...)
	if err != nil {
		return err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if n < 1 {
		return ErrRecordNotFound
	}
	return nil
}

func (s *Store) UpdateFields(ctx context.Context, table string, cond Cond, update map[string]interface{}) (int64, error) {
	sql, values, err := builder.BuildUpdate(table, map[string]interface{}(cond), update)
	if err != nil {
		return 0, err
	}
	result, err := s.db.ExecContext(ctx, sql, values...)
	if err != nil {
		return 0, err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return 0, err
	}
	return n, nil
}

func (s *Store) Count(ctx context.Context, table string, cond Cond) (uint64, error) {
	aggr := builder.AggregateCount("1")
	rr, err := builder.AggregateQuery(ctx, s.db.Raw().Unsafe().DB, table, map[string]interface{}(cond), aggr)
	if err != nil {
		return 0, err
	}
	return uint64(rr.Int64()), nil
}
