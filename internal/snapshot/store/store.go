package store

import (
	"context"

	"github.com/shuaibizhang/codecoverage/store"
)

type SnapshotStore interface {
	Save(ctx context.Context, info *SnapshotInfo) error
	QueryBySnapshotID(ctx context.Context, snapshotID string) (*SnapshotInfo, error)
	QueryLatestByCommit(ctx context.Context, module, branch, commit string) (*SnapshotInfo, error)
	ListByCommit(ctx context.Context, module, branch, commit string) ([]*SnapshotInfo, error)
	GetMetadataList(ctx context.Context, module, branch string) ([]string, []string, []string, error)
}

type snapshotStore struct {
	store *store.Store
}

func NewSnapshotStore(s *store.Store) SnapshotStore {
	return &snapshotStore{
		store: s,
	}
}

func (s *snapshotStore) Save(ctx context.Context, info *SnapshotInfo) error {
	id, err := s.store.SaveRecord(ctx, info.TableName(), info.ID, info)
	if err != nil {
		return err
	}
	info.ID = id
	return nil
}

func (s *snapshotStore) QueryBySnapshotID(ctx context.Context, snapshotID string) (*SnapshotInfo, error) {
	cond := store.NewCond().
		Where("snapshot_id", snapshotID).
		NotDeleted()

	var infos []*SnapshotInfo
	err := s.store.Query(ctx, SnapshotInfo{}.TableName(), cond, nil, &infos)
	if err != nil {
		return nil, err
	}

	if len(infos) == 0 {
		return nil, store.ErrRecordNotFound
	}

	return infos[0], nil
}

func (s *snapshotStore) QueryLatestByCommit(ctx context.Context, module, branch, commit string) (*SnapshotInfo, error) {
	cond := store.NewCond().
		Where("module", module).
		Where("branch", branch).
		Where("commit", commit).
		NotDeleted().
		OrderBy("_created_time DESC")

	var infos []*SnapshotInfo
	err := s.store.Query(ctx, SnapshotInfo{}.TableName(), cond, nil, &infos)
	if err != nil {
		return nil, err
	}

	if len(infos) == 0 {
		return nil, store.ErrRecordNotFound
	}

	return infos[0], nil
}

func (s *snapshotStore) ListByCommit(ctx context.Context, module, branch, commit string) ([]*SnapshotInfo, error) {
	cond := store.NewCond().
		Where("module", module).
		Where("branch", branch).
		Where("commit", commit).
		NotDeleted().
		OrderBy("_created_time DESC")

	var infos []*SnapshotInfo
	err := s.store.Query(ctx, SnapshotInfo{}.TableName(), cond, nil, &infos)
	if err != nil {
		return nil, err
	}

	return infos, nil
}

func (s *snapshotStore) GetMetadataList(ctx context.Context, module, branch string) ([]string, []string, []string, error) {
	type moduleItem struct {
		Module string `ddb:"module"`
	}
	type branchItem struct {
		Branch string `ddb:"branch"`
	}
	type commitItem struct {
		Commit string `ddb:"commit"`
	}

	cond := store.NewCond().NotDeleted()
	if module != "" {
		cond = cond.Where("module", module)
	}

	var mItems []moduleItem
	err := s.store.Query(ctx, SnapshotInfo{}.TableName(), cond, []string{"DISTINCT module as module"}, &mItems)
	if err != nil {
		return nil, nil, nil, err
	}
	modules := make([]string, 0, len(mItems))
	for _, item := range mItems {
		modules = append(modules, item.Module)
	}

	var bItems []branchItem
	bCond := store.NewCond()
	for k, v := range cond {
		bCond[k] = v
	}
	err = s.store.Query(ctx, SnapshotInfo{}.TableName(), bCond.GroupBy("branch").OrderBy("MAX(_created_time) DESC"), []string{"branch", "MAX(_created_time)"}, &bItems)
	if err != nil {
		return nil, nil, nil, err
	}
	branches := make([]string, 0, len(bItems))
	for _, item := range bItems {
		branches = append(branches, item.Branch)
	}

	var cItems []commitItem
	cCond := store.NewCond()
	for k, v := range cond {
		cCond[k] = v
	}
	if branch != "" {
		cCond = cCond.Where("branch", branch)
	}
	err = s.store.Query(ctx, SnapshotInfo{}.TableName(), cCond.GroupBy("commit").OrderBy("MAX(_created_time) DESC"), []string{"commit", "MAX(_created_time)"}, &cItems)
	if err != nil {
		return nil, nil, nil, err
	}
	commits := make([]string, 0, len(cItems))
	for _, item := range cItems {
		commits = append(commits, item.Commit)
	}

	return modules, branches, commits, nil
}
