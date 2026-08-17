package store

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

type Store struct {
	DB          *sql.DB
	processLock *ProcessLock
}

func Open(dataDir string) (*Store, error) {
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		return nil, fmt.Errorf("mkdir data dir: %w", err)
	}
	processLock, err := acquireDaemonProcessLock(dataDir)
	if err != nil {
		return nil, err
	}
	dbPath := filepath.Join(dataDir, "forge.sqlite")
	dsn := fmt.Sprintf("file:%s?_pragma=foreign_keys(1)&_pragma=journal_mode(WAL)", dbPath)
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		_ = processLock.Close()
		return nil, err
	}
	db.SetMaxOpenConns(1)
	if err := migrate(db); err != nil {
		_ = db.Close()
		_ = processLock.Close()
		return nil, err
	}
	return &Store{DB: db, processLock: processLock}, nil
}

func (s *Store) Close() error {
	if s.DB == nil {
		return nil
	}
	err := s.DB.Close()
	if s.processLock != nil {
		err = errors.Join(err, s.processLock.Close())
		s.processLock = nil
	}
	return err
}
