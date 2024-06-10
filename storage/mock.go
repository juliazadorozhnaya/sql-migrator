package storage

import (
	"context"
)

type MockSqlStorage struct {
	migrations []IMigration
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
	m.migrations = append(m.migrations, migration)
	return nil
}

func (m *MockSqlStorage) Migrate(ctx context.Context, sql string) error {
	return nil
}

func (m *MockSqlStorage) SelectMigrations(ctx context.Context) ([]IMigration, error) {
	return m.migrations, nil
}

func (m *MockSqlStorage) SelectLastMigrationByStatus(ctx context.Context, status string) (IMigration, error) {
	if len(m.migrations) == 0 {
		return nil, ErrMigrationNotFound
	}

	for i := len(m.migrations) - 1; i >= 0; i-- {
		if m.migrations[i].GetStatus() == status {
			return m.migrations[i], nil
		}
	}

	return nil, ErrMigrationNotFound
}

func (m *MockSqlStorage) DeleteMigrations(ctx context.Context) error {
	m.migrations = []IMigration{}
	return nil
}
