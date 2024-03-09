package store

import (
	"sync"
	"time"

	"github.com/rs/zerolog/log"
)

type Value struct {
	Value   string    `json:"value"`
	Updated time.Time `json:"updated"`
}

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
	s.data.Store(key, Value{
		Value:   value,
		Updated: time.Now(),
	})
}

func (s *KeyValueStore) Export() map[string]Value {
	log.Debug().Msg("KeyValueStoreExport")

	export := map[string]Value{}

	s.data.Range(func(key, value interface{}) bool {
		export[key.(string)] = value.(Value)
		return true
	})

	return export
}

func (s *KeyValueStore) MergeIntoStore(remote map[string]Value) {
	log.Debug().Msg("Merge")
	for key, remotevalue := range remote {
		localvalue, ok := s.data.Load(key)
		if !ok {
			log.Info().Str("key", key).Msg("value not found, merging into local state")
			s.data.Store(key, remotevalue)
		} else {
			// failure/race with broadcast update ?
			if remotevalue.Updated.After(localvalue.(Value).Updated) {
				log.Info().Str("key", key).Msg("value not found, merging into local state")
				s.data.Store(key, remotevalue)
			}
		}
		// keys present locally but not remotely , or more recent local keys should be catered
		// for by other nodes pulling state
	}
}

func (s *KeyValueStore) Get(key string) (string, bool) {
	log.Debug().Str("key", key).Msg("KeyValueStoreGet")
	value, ok := s.data.Load(key)
	if !ok {
		log.Info().Str("key", key).Msg("value not found")
		return "", false
	}
	return value.(Value).Value, true
}

func (s *KeyValueStore) Delete(key string) {
	log.Debug().Str("key", key).Msg("KeyValueStoreDelete")
	s.data.Delete(key)
}
