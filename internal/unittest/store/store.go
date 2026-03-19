package store

import (
	"context"

	"github.com/shuaibizhang/codecoverage/store"
)

type UnitTestStore interface {
	Query(ctx context.Context, module, branch, commit string) (*UnittestTask, error)
	Save(ctx context.Context, task *UnittestTask) error
}

type unitTestStore struct {
	store *store.Store
}

func NewUnitTestStore(s *store.Store) UnitTestStore {
	return &unitTestStore{
		store: s,
	}
}

func (s *unitTestStore) Query(ctx context.Context, module, branch, commit string) (*UnittestTask, error) {
	cond := store.NewCond().
		Where("module", module).
		Where("branch", branch).
		Where("commit", commit).
		NotDeleted()

	var tasks []*UnittestTask
	err := s.store.Query(ctx, UnittestTask{}.TableName(), cond, nil, &tasks)
	if err != nil {
		return nil, err
	}

	if len(tasks) == 0 {
		return nil, store.ErrRecordNotFound
	}

	return tasks[0], nil
}

func (s *unitTestStore) Save(ctx context.Context, task *UnittestTask) error {
	id, err := s.store.SaveRecord(ctx, task.TableName(), task.ID, task)
	if err != nil {
		return err
	}
	task.ID = id
	return nil
}
