package reportlock

import (
	"context"
	"errors"
	"testing"
)

// mockRWLock 实现 dislock.RWLock 接口用于测试
type mockRWLock struct {
	lockErr    error
	unlockErr  error
	canWrite   bool
	lockCalled bool
	writeMode  bool
}

func (m *mockRWLock) Lock(ctx context.Context, write bool) error {
	m.lockCalled = true
	m.writeMode = write
	return m.lockErr
}

func (m *mockRWLock) Unlock(ctx context.Context) error {
	return m.unlockErr
}

func (m *mockRWLock) CanWrite() bool {
	return m.canWrite
}

func (m *mockRWLock) Clean(ctx context.Context) (int, error) {
	return 0, nil
}

func TestReportLock(t *testing.T) {
	ctx := context.Background()

	t.Run("Lock_Success", func(t *testing.T) {
		mock := &mockRWLock{}
		rl := &reportLock{rwLock: mock}

		err := rl.Lock(ctx)
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
		if !mock.lockCalled {
			t.Error("expected Lock to be called")
		}
		if !mock.writeMode {
			t.Error("expected Lock to be called with write=true")
		}
	})

	t.Run("Lock_Error", func(t *testing.T) {
		expectedErr := errors.New("lock error")
		mock := &mockRWLock{lockErr: expectedErr}
		rl := &reportLock{rwLock: mock}

		err := rl.Lock(ctx)
		if err != expectedErr {
			t.Errorf("expected error %v, got %v", expectedErr, err)
		}
	})

	t.Run("Unlock_Success", func(t *testing.T) {
		mock := &mockRWLock{}
		rl := &reportLock{rwLock: mock}

		err := rl.Unlock(ctx)
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
	})

	t.Run("Unlock_Error", func(t *testing.T) {
		expectedErr := errors.New("unlock error")
		mock := &mockRWLock{unlockErr: expectedErr}
		rl := &reportLock{rwLock: mock}

		err := rl.Unlock(ctx)
		if err != expectedErr {
			t.Errorf("expected error %v, got %v", expectedErr, err)
		}
	})

	t.Run("CanWrite", func(t *testing.T) {
		mock := &mockRWLock{canWrite: true}
		rl := &reportLock{rwLock: mock}

		if !rl.CanWrite(ctx) {
			t.Error("expected CanWrite to return true")
		}

		mock.canWrite = false
		if rl.CanWrite(ctx) {
			t.Error("expected CanWrite to return false")
		}
	})
}

func TestNewReportLock(t *testing.T) {
	// 简单验证 NewReportLock 不会 panic
	// 注意：由于它内部调用了 dislock.NewRWLock，我们无法在这里轻松验证其内部状态
	// 但我们可以确保它返回了一个有效的 ReportLock 接口实现
	rl := NewReportLock(nil, nil, "test_lock_key")
	if rl == nil {
		t.Fatal("expected NewReportLock to return non-nil")
	}
}
