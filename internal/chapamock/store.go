package chapamock

import (
	"context"
	"encoding/json"
	"errors"
	"sort"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	recordTTL   = 30 * 24 * time.Hour
	recentLimit = 200
)

// store keeps payments and transfers as JSON records. Redis lets them survive API
// restarts during development; tests use the in-memory version.
type store interface {
	put(ctx context.Context, kind, id string, v any) error
	// get reports false when the record doesn't exist.
	get(ctx context.Context, kind, id string, v any) (bool, error)
	// recent lists record IDs, newest first.
	recent(ctx context.Context, kind string, n int) ([]string, error)
}

type redisStore struct {
	rdb redis.Cmdable
}

func (s redisStore) key(kind, id string) string { return "mockchapa:" + kind + ":" + id }

func (s redisStore) put(ctx context.Context, kind, id string, v any) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	index := "mockchapa:" + kind
	pipe := s.rdb.TxPipeline()
	pipe.Set(ctx, s.key(kind, id), data, recordTTL)
	pipe.ZAddNX(ctx, index, redis.Z{Score: float64(time.Now().UnixNano()), Member: id})
	pipe.ZRemRangeByRank(ctx, index, 0, -recentLimit-1)
	pipe.Expire(ctx, index, recordTTL)
	_, err = pipe.Exec(ctx)
	return err
}

func (s redisStore) get(ctx context.Context, kind, id string, v any) (bool, error) {
	data, err := s.rdb.Get(ctx, s.key(kind, id)).Bytes()
	if errors.Is(err, redis.Nil) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, json.Unmarshal(data, v)
}

func (s redisStore) recent(ctx context.Context, kind string, n int) ([]string, error) {
	return s.rdb.ZRevRange(ctx, "mockchapa:"+kind, 0, int64(n-1)).Result()
}

type memoryStore struct {
	mu      sync.Mutex
	records map[string][]byte
	added   map[string]time.Time
}

func newMemoryStore() *memoryStore {
	return &memoryStore{records: map[string][]byte{}, added: map[string]time.Time{}}
}

func (s *memoryStore) put(_ context.Context, kind, id string, v any) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	key := kind + ":" + id
	if _, ok := s.added[key]; !ok {
		s.added[key] = time.Now()
	}
	s.records[key] = data
	return nil
}

func (s *memoryStore) get(_ context.Context, kind, id string, v any) (bool, error) {
	s.mu.Lock()
	data, ok := s.records[kind+":"+id]
	s.mu.Unlock()
	if !ok {
		return false, nil
	}
	return true, json.Unmarshal(data, v)
}

func (s *memoryStore) recent(_ context.Context, kind string, n int) ([]string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	type entry struct {
		id string
		at time.Time
	}
	var entries []entry
	prefix := kind + ":"
	for key, at := range s.added {
		if len(key) > len(prefix) && key[:len(prefix)] == prefix {
			entries = append(entries, entry{key[len(prefix):], at})
		}
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].at.After(entries[j].at) })
	ids := make([]string, 0, min(n, len(entries)))
	for i := 0; i < len(entries) && i < n; i++ {
		ids = append(ids, entries[i].id)
	}
	return ids, nil
}
