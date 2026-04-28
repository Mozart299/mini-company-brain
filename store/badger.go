// BadgerDB implementation of the Store interface.
// BadgerDB is an embeddable key-value store — each store node runs its own instance
// backed by files in DATA_DIR, giving us real persistence unlike the MemoryStore.
package store

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"company-brain/pkg/types"

	badger "github.com/dgraph-io/badger/v4"
)

type BadgerStore struct {
	db *badger.DB
}

func NewBadgerStore(dir string) (*BadgerStore, error) {
	opts := badger.DefaultOptions(dir).WithLogger(nil)
	db, err := badger.Open(opts)
	if err != nil {
		return nil, fmt.Errorf("open badger at %s: %w", dir, err)
	}
	return &BadgerStore{db: db}, nil
}

func (b *BadgerStore) Close() error {
	return b.db.Close()
}

func (b *BadgerStore) Set(_ context.Context, fact types.Fact) error {
	fact.Version = time.Now().UnixNano()
	fact.CreatedAt = time.Now()
	data, err := json.Marshal(fact)
	if err != nil {
		return fmt.Errorf("marshal fact: %w", err)
	}
	return b.db.Update(func(txn *badger.Txn) error {
		return txn.Set([]byte(fact.Key), data)
	})
}

func (b *BadgerStore) Get(_ context.Context, key string) (types.Fact, error) {
	var fact types.Fact
	err := b.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte(key))
		if err != nil {
			return err
		}
		return item.Value(func(val []byte) error {
			return json.Unmarshal(val, &fact)
		})
	})
	if err == badger.ErrKeyNotFound {
		return types.Fact{}, fmt.Errorf("key not found: %s", key)
	}
	return fact, err
}

func (b *BadgerStore) Delete(_ context.Context, key string) error {
	return b.db.Update(func(txn *badger.Txn) error {
		return txn.Delete([]byte(key))
	})
}

func (b *BadgerStore) List(_ context.Context, prefix string) ([]types.Fact, error) {
	var facts []types.Fact
	err := b.db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		if prefix != "" {
			opts.Prefix = []byte(prefix)
		}
		it := txn.NewIterator(opts)
		defer it.Close()
		for it.Rewind(); it.Valid(); it.Next() {
			if err := it.Item().Value(func(val []byte) error {
				var f types.Fact
				if err := json.Unmarshal(val, &f); err != nil {
					return err
				}
				facts = append(facts, f)
				return nil
			}); err != nil {
				return err
			}
		}
		return nil
	})
	return facts, err
}
