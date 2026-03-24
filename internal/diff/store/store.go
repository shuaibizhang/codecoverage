package store

import (
	"context"

	"github.com/shuaibizhang/codecoverage/store"
)

type DiffStore interface {
	Query(ctx context.Context, module, commitID, baseCommitID string) (*DiffCache, error)
	Save(ctx context.Context, cache *DiffCache) error
}

type diffStore struct {
	store *store.Store
}

func NewDiffStore(s *store.Store) DiffStore {
	return &diffStore{
		store: s,
	}
}

func (s *diffStore) Query(ctx context.Context, module, commitID, baseCommitID string) (*DiffCache, error) {
	cond := store.NewCond().
		Where("module", module).
		Where("commit_id", commitID).
		Where("base_commit_id", baseCommitID).
		NotDeleted()

	var caches []*DiffCache
	err := s.store.Query(ctx, DiffCache{}.TableName(), cond, nil, &caches)
	if err != nil {
		return nil, err
	}

	if len(caches) == 0 {
		return nil, store.ErrRecordNotFound
	}

	return caches[0], nil
}

func (s *diffStore) Save(ctx context.Context, cache *DiffCache) error {
	id, err := s.store.SaveRecord(ctx, cache.TableName(), cache.ID, cache)
	if err != nil {
		return err
	}
	cache.ID = id
	return nil
}
