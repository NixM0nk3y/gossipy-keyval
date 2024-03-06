package store

import (
	"sync"

	"github.com/rs/zerolog/log"
)

type KeyValueStore struct {
	data sync.Map
}

func NewKeyValueStore() *KeyValueStore {
	log.Debug().Msg("KeyValueStoreNew")
	return &KeyValueStore{
		data: sync.Map{},
	}
}

func (s *KeyValueStore) Set(key string, value string) {
	log.Debug().Str("key", key).Str("value", value).Msg("KeyValueStoreSet")
	s.data.Store(key, value)
}

func (s *KeyValueStore) Get(key string) (string, bool) {
	log.Debug().Str("key", key).Msg("KeyValueStoreGet")
	value, ok := s.data.Load(key)
	if !ok {
		log.Info().Str("key", key).Msg("value not found")
		return "", false
	}
	return value.(string), true
}

func (s *KeyValueStore) Delete(key string) {
	log.Debug().Str("key", key).Msg("KeyValueStoreDelete")
	s.data.Delete(key)
}
