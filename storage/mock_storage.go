package storage

import (
	"context"
	"errors"
)

type MockSqlStorage struct {
	migrations []IMigration
}

func NewMockSqlStorage() *MockSqlStorage {
	return &MockSqlStorage{
		migrations: []IMigration{},
	}
}

func (m *MockSqlStorage) Connect(ctx context.Context) error {
	return nil
}

func (m *MockSqlStorage) Close() error {
	return nil
}

func (m *MockSqlStorage) Lock(ctx context.Context) error {
	return nil
}

func (m *MockSqlStorage) Unlock(ctx context.Context) error {
	return nil
}

func (m *MockSqlStorage) InsertMigration(ctx context.Context, migration IMigration) error {
	for _, m := range m.migrations {
		if m.GetVersion() == migration.GetVersion() && m.GetName() == migration.GetName() {
			m.SetStatus(migration.GetStatus())
			m.SetStatusChangeTime(migration.GetStatusChangeTime())
			m.SetVersion(migration.GetVersion())
			m.SetName(migration.GetName())
			return nil
		}
	}
	m.migrations = append(m.migrations, migration)
	return nil
}

func (m *MockSqlStorage) UpdateMigration(ctx context.Context, migration IMigration) error {
	for _, m := range m.migrations {
		if m.GetVersion() == migration.GetVersion() && m.GetName() == migration.GetName() {
			m.SetStatus(migration.GetStatus())
			m.SetStatusChangeTime(migration.GetStatusChangeTime())
			return nil
		}
	}
	return errors.New("migration not found")
}

func (m *MockSqlStorage) Migrate(ctx context.Context, sql string) error {
	return nil
}

func (m *MockSqlStorage) SelectMigrations(ctx context.Context) ([]IMigration, error) {
	if len(m.migrations) == 0 {
		return nil, errors.New("no migrations found")
	}
	return m.migrations, nil
}

func (m *MockSqlStorage) SelectLastMigrationByStatus(ctx context.Context, status string) (IMigration, error) {
	for i := len(m.migrations) - 1; i >= 0; i-- {
		if m.migrations[i].GetStatus() == status {
			return m.migrations[i], nil
		}
	}
	return nil, errors.New("no migrations found with status " + status)
}

func (m *MockSqlStorage) DeleteMigrations(ctx context.Context) error {
	m.migrations = []IMigration{}
	return nil
}
