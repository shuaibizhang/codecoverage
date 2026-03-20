package store

import (
	"context"

	"github.com/shuaibizhang/codecoverage/store"
)

type UnitTestStore interface {
	Query(ctx context.Context, module, branch, commit string) (*UnittestTask, error)
	Save(ctx context.Context, task *UnittestTask) error
	GetMetadataList(ctx context.Context) ([]string, []string, []string, error)
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

func (s *unitTestStore) GetMetadataList(ctx context.Context) ([]string, []string, []string, error) {
	// 定义辅助结构体以便 scanner 扫描
	type moduleItem struct {
		Module string `ddb:"module"`
	}
	type branchItem struct {
		Branch string `ddb:"branch"`
	}
	type commitItem struct {
		Commit string `ddb:"commit"`
	}

	// 获取去重后的 module
	var mItems []moduleItem
	err := s.store.Query(ctx, UnittestTask{}.TableName(), store.NewCond().NotDeleted(), []string{"DISTINCT module as module"}, &mItems)
	if err != nil {
		return nil, nil, nil, err
	}
	modules := make([]string, 0, len(mItems))
	for _, item := range mItems {
		modules = append(modules, item.Module)
	}

	// 获取去重后的 branch
	var bItems []branchItem
	err = s.store.Query(ctx, UnittestTask{}.TableName(), store.NewCond().NotDeleted(), []string{"DISTINCT branch as branch"}, &bItems)
	if err != nil {
		return nil, nil, nil, err
	}
	branches := make([]string, 0, len(bItems))
	for _, item := range bItems {
		branches = append(branches, item.Branch)
	}

	// 获取去重后的 commit
	var cItems []commitItem
	err = s.store.Query(ctx, UnittestTask{}.TableName(), store.NewCond().NotDeleted(), []string{"DISTINCT commit as commit"}, &cItems)
	if err != nil {
		return nil, nil, nil, err
	}
	commits := make([]string, 0, len(cItems))
	for _, item := range cItems {
		commits = append(commits, item.Commit)
	}

	return modules, branches, commits, nil
}
