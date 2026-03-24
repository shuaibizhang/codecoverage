package store

import (
	"context"

	"github.com/shuaibizhang/codecoverage/store"
)

type UnitTestStore interface {
	Query(ctx context.Context, module, branch, commit string) (*UnittestTask, error)
	Save(ctx context.Context, task *UnittestTask) error
	GetMetadataList(ctx context.Context, module, branch string) ([]string, []string, []string, error)
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

func (s *unitTestStore) GetMetadataList(ctx context.Context, module, branch string) ([]string, []string, []string, error) {
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

	// 基础查询条件
	cond := store.NewCond().NotDeleted()

	// 如果有传入 module，先过滤基础条件，确保所有后续查询都锁定在该模块
	if module != "" {
		// 精确匹配模块名
		cond = cond.Where("module", module)
	}

	// 获取该模块下的 module 列表（通常只有一个，或者如果为空则获取所有）
	var mItems []moduleItem
	err := s.store.Query(ctx, UnittestTask{}.TableName(), cond, []string{"DISTINCT module as module"}, &mItems)
	if err != nil {
		return nil, nil, nil, err
	}
	modules := make([]string, 0, len(mItems))
	for _, item := range mItems {
		modules = append(modules, item.Module)
	}

	// 获取该模块下去重后的 branch，按最新创建时间排序
	var bItems []branchItem
	// 创建新的 cond 副本，避免 GroupBy/OrderBy 相互影响
	bCond := store.NewCond()
	for k, v := range cond {
		bCond[k] = v
	}
	err = s.store.Query(ctx, UnittestTask{}.TableName(), bCond.GroupBy("branch").OrderBy("MAX(_created_time) DESC"), []string{"branch", "MAX(_created_time)"}, &bItems)
	if err != nil {
		return nil, nil, nil, err
	}
	branches := make([]string, 0, len(bItems))
	for _, item := range bItems {
		branches = append(branches, item.Branch)
	}

	// 获取该模块下去重后的 commit，按最新创建时间排序
	var cItems []commitItem
	cCond := store.NewCond()
	for k, v := range cond {
		cCond[k] = v
	}
	// 如果传入了 branch，则进一步过滤 commit
	if branch != "" {
		cCond = cCond.Where("branch", branch)
	}
	err = s.store.Query(ctx, UnittestTask{}.TableName(), cCond.GroupBy("commit").OrderBy("MAX(_created_time) DESC"), []string{"commit", "MAX(_created_time)"}, &cItems)
	if err != nil {
		return nil, nil, nil, err
	}
	commits := make([]string, 0, len(cItems))
	for _, item := range cItems {
		commits = append(commits, item.Commit)
	}

	return modules, branches, commits, nil
}
