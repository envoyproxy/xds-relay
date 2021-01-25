package cache

import (
	"sync"

	"github.com/envoyproxy/xds-relay/internal/app/transport"
)

type RequestsStore struct {
	m *sync.Map
}

func NewRequestsStore() *RequestsStore {
	return &RequestsStore{
		m: &sync.Map{},
	}
}

func (m *RequestsStore) Set(r transport.Request) {
	_, _ = m.m.LoadOrStore(r, struct{}{})
}

func (m *RequestsStore) get(r transport.Request) bool {
	_, ok := m.m.Load(r)
	return ok
}

func (m *RequestsStore) Delete(r transport.Request) {
	m.m.Delete((r))
}

func (m *RequestsStore) ForEach(f func(key transport.Request)) {
	m.m.Range(func(key, value interface{}) bool {
		watch, ok := key.(transport.Request)
		if !ok {
			panic(key)
		}
		f(watch)
		return true
	})
}
