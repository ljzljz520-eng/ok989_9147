package auth

import (
	"context"
	"errors"
	"internalauth/internal/domain"
	"sync"
)

type MemoryRedis struct {
	mu     sync.RWMutex
	values map[string]bool
	fail   map[string]error
}

func NewMemoryRedis() *MemoryRedis {
	return &MemoryRedis{values: map[string]bool{}, fail: map[string]error{}}
}
func (r *MemoryRedis) Put(configID, key string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.values = configured(r.values)
	r.values[configID+"\x00"+key] = true
}
func configured(v map[string]bool) map[string]bool {
	if v == nil {
		return map[string]bool{}
	}
	return v
}
func (r *MemoryRedis) SetFailure(configID string, e error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.fail == nil {
		r.fail = map[string]error{}
	}
	r.fail[configID] = e
}
func (r *MemoryRedis) Exists(_ context.Context, c domain.RedisConfig, key string) (bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if e := r.fail[c.ID]; e != nil {
		return false, e
	}
	return r.values[c.ID+"\x00"+key], nil
}
func (_ *MemoryRedis) Ping(_ context.Context, _ domain.RedisConfig) error { return nil }
func (_ *MemoryRedis) Close() error                                       { return errors.New("memory redis does not close") }
