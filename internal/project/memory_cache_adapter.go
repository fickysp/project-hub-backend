package project

import (
	"context"
	"fmt"
	"strings"
	"time"

	memory "github.com/patrickmn/go-cache"
)

type MemoryCacheAdapter struct {
	store *memory.Cache
}

var _ CacheClient = (*MemoryCacheAdapter)(nil)

func NewMemoryCacheAdapter(store *memory.Cache) *MemoryCacheAdapter {
	return &MemoryCacheAdapter{store: store}
}

func (m *MemoryCacheAdapter) Get(
	ctx context.Context,
	key string,
) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	value, found := m.store.Get(key)
	if !found {
		return nil, nil
	}

	data, ok := value.([]byte)
	if !ok {
		return nil, fmt.Errorf("cache value for %q is not []byte", key)
	}

	return append([]byte(nil), data...), nil
}

func (m *MemoryCacheAdapter) Set(
	ctx context.Context,
	key string,
	value []byte,
	ttl time.Duration,
) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	m.store.Set(key, append([]byte(nil), value...), ttl)
	return nil
}

func (m *MemoryCacheAdapter) DeletePrefix(
	ctx context.Context,
	prefix string,
) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	for key := range m.store.Items() {
		if strings.HasPrefix(key, prefix) {
			m.store.Delete(key)
		}
	}

	return nil
}
