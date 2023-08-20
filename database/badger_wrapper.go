package database

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/dgraph-io/badger/v4"
)

const (
	// Default BadgerDB discardRatio. It represents the discard ratio for the
	// BadgerDB GC.
	//
	// Ref: https://godoc.org/github.com/dgraph-io/badger#DB.RunValueLogGC
	badgerDiscardRatio = 0.5

	// Default BadgerDB GC interval
	badgerGCInterval = 10 * time.Minute
)

// BadgerAlertNamespace defines the alerts BadgerDB namespace.
var BadgerAlertNamespace = []byte("alerts")

type (
	// DBInstance defines an embedded key/value store database interface.
	DBInstance interface {
		Get(namespace, key []byte) (value string, err error)
		Update(namespace, key, value []byte) error
		Insert(namespace, key, value []byte) error
		Has(namespace, key []byte) (bool, error)
		Close() error
	}

	// BadgerDB is a wrapper around a BadgerDB backend database that implements
	// the DB interface.
	BadgerDB struct {
		db         *badger.DB
		ctx        context.Context
		cancelFunc context.CancelFunc
	}
)

// NewBadgerDB returns a new initialized BadgerDB database implementing the DB
// interface. If the database cannot be initialized, an error will be returned.
func NewBadgerDB(dataDir string) (DBInstance, error) {
	if err := os.MkdirAll(dataDir, 0o774); err != nil {
		return nil, err
	}

	opts := badger.DefaultOptions(dataDir)
	opts.Logger = nil
	opts.SyncWrites = true

	log.Printf("Opening db with these options: %+v\n", opts)

	badgerDB, err := badger.Open(opts)
	if err != nil {
		return nil, err
	}

	bdb := &BadgerDB{
		db: badgerDB,
	}
	bdb.ctx, bdb.cancelFunc = context.WithCancel(context.Background())

	go bdb.runGC()
	return bdb, nil
}

// Get implements the DB interface. It attempts to get a value for a given key
// and namespace. If the key does not exist in the provided namespace, an error
// is returned, otherwise the retrieved value.
func (bdb *BadgerDB) Get(namespace, key []byte) (value string, err error) {
	err = bdb.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get(badgerNamespaceKey(namespace, key))
		if err != nil {
			return err
		}
		// log.Println("Get - View: ", item)
		err = item.Value(func(val []byte) error {
			value = string(val)
			// log.Println("Get - View - Val: ", string(val))
			// log.Println("Get - View - value 1: ", value)
			return nil
		})
		if err != nil {
			return err
		}
		// log.Println("Get - View - Value 2: ", value)
		return nil
	})
	return
}

// Update implements the DB interface. It attempts to store or update a value for a given key
// and namespace. If the key/value pair cannot be saved, an error is returned.
func (bdb *BadgerDB) Update(namespace, key, value []byte) error {
	err := bdb.db.Update(func(txn *badger.Txn) error {
		return txn.Set(badgerNamespaceKey(namespace, key), value)
	})
	if err != nil {
		log.Printf("failed to set key %s for namespace %s: %v", key, namespace, err)
		return err
	}

	return nil
}

// Insert implements the DB interface. It attempts to store a unique key
// and namespace. If the key/value pair cannot be saved, an error is returned.
func (bdb *BadgerDB) Insert(namespace, key, value []byte) error {
	has, err := bdb.Has(namespace, key)
	if err != nil {
		log.Printf("error checking if exists the key %s for namespace %s: %v", key, namespace, err)
		return err
	}
	if has {
		err = fmt.Errorf("error: already exists the key %s for namespace %s", key, namespace)
		return err
	}
	err = bdb.db.Update(func(txn *badger.Txn) error {
		return txn.Set(badgerNamespaceKey(namespace, key), value)
	})
	if err != nil {
		log.Printf("failed to set key %s for namespace %s: %v", key, namespace, err)
		return err
	}
	return nil
}

// Has implements the DB interface. It returns a boolean reflecting if the
// datbase has a given key for a namespace or not. An error is only returned if
// an error to Get would be returned that is not of type badger.ErrKeyNotFound.
func (bdb *BadgerDB) Has(namespace, key []byte) (ok bool, err error) {
	_, errGet := bdb.Get(namespace, key)
	switch errGet {
	case badger.ErrKeyNotFound:
		ok, err = false, nil
	case nil:
		ok, err = true, nil
	default:
		ok, err = false, errGet
	}

	return
}

// Close implements the DB interface. It closes the connection to the underlying
// BadgerDB database as well as invoking the context's cancel function.
func (bdb *BadgerDB) Close() error {
	bdb.cancelFunc()
	return bdb.db.Close()
}

// runGC triggers the garbage collection for the BadgerDB backend database. It
// should be run in a goroutine.
func (bdb *BadgerDB) runGC() {
	ticker := time.NewTicker(badgerGCInterval)
	for {
		select {
		case <-ticker.C:
			err := bdb.db.RunValueLogGC(badgerDiscardRatio)
			if err != nil {
				// don't report error when GC didn't result in any cleanup
				if err == badger.ErrNoRewrite {
					log.Printf("no BadgerDB GC occurred: %v", err)
				} else {
					log.Printf("failed to GC BadgerDB: %v", err)
				}
			}

		case <-bdb.ctx.Done():
			return
		}
	}
}

// badgerNamespaceKey returns a composite key used for lookup and storage for a
// given namespace and key.
func badgerNamespaceKey(namespace, key []byte) []byte {
	prefix := []byte(fmt.Sprintf("%s/", namespace))
	return append(prefix, key...)
}
