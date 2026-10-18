// Package testutil 提供可回滚的内存仓储，供不依赖外部 MySQL 的业务和 HTTP 测试使用。
package testutil

import (
	"context"
	"fmt"
	"sync"

	"dailyWins-service/internal/biz"
	e "dailyWins-service/internal/errors"
)

type Store struct {
	mu         sync.Mutex
	Accounts   map[int32]*biz.Account
	Sessions   map[string]*biz.Session
	Tasks      map[string]*biz.Task
	Operations map[string]*biz.TaskOperation
}

func NewStore() *Store {
	return &Store{Accounts: map[int32]*biz.Account{}, Sessions: map[string]*biz.Session{}, Tasks: map[string]*biz.Task{}, Operations: map[string]*biz.TaskOperation{}}
}
func (s *Store) ExecTx(ctx context.Context, fn func(context.Context) error) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	tasks := map[string]*biz.Task{}
	for k, v := range s.Tasks {
		c := *v
		tasks[k] = &c
	}
	ops := map[string]*biz.TaskOperation{}
	for k, v := range s.Operations {
		ops[k] = v
	}
	if err := fn(ctx); err != nil {
		s.Tasks = tasks
		s.Operations = ops
		return err
	}
	return nil
}
func (s *Store) LockOwner(_ context.Context, id int32) error {
	if s.Accounts[id] == nil {
		return e.E(e.ErrUnauthorized)
	}
	return nil
}
func (s *Store) GetTask(_ context.Context, u int32, id string) (*biz.Task, error) {
	t := s.Tasks[id]
	if t == nil || t.UserID != u {
		return nil, nil
	}
	c := *t
	return &c, nil
}
func (s *Store) ListTasks(_ context.Context, u int32, from, to string) ([]*biz.Task, error) {
	var result []*biz.Task
	for _, t := range s.Tasks {
		if t.UserID == u && !t.Deleted && t.Date >= from && t.Date <= to {
			c := *t
			result = append(result, &c)
		}
	}
	return result, nil
}
func (s *Store) SaveTask(_ context.Context, t *biz.Task) error {
	if old := s.Tasks[t.ID]; old != nil && old.UserID != t.UserID {
		return e.E(e.ErrDuplicate)
	}
	c := *t
	s.Tasks[t.ID] = &c
	return nil
}
func opKey(u int32, id string) string { return fmt.Sprintf("%d:%s", u, id) }
func (s *Store) GetOperation(_ context.Context, u int32, id string) (*biz.TaskOperation, error) {
	return s.Operations[opKey(u, id)], nil
}
func (s *Store) SaveOperation(_ context.Context, u int32, id string, op *biz.TaskOperation) error {
	s.Operations[opKey(u, id)] = op
	return nil
}
func (s *Store) FindAccount(_ context.Context, name string) (*biz.Account, error) {
	for _, a := range s.Accounts {
		if a.Username == name {
			return a, nil
		}
	}
	return nil, nil
}
func (s *Store) GetAccount(_ context.Context, u int32) (*biz.Account, error) {
	return s.Accounts[u], nil
}
func (s *Store) CreateSession(_ context.Context, v *biz.Session) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Sessions[v.ID] = v
	return nil
}
func (s *Store) FindSession(_ context.Context, id string) (*biz.Session, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.Sessions[id], nil
}
func (s *Store) RevokeSession(_ context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.Sessions, id)
	return nil
}
