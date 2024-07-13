package app

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/juliazadorozhnaya/sql-migrator/logger"
	"github.com/juliazadorozhnaya/sql-migrator/storage"
	"github.com/stretchr/testify/assert"
)

func TestCreateMigrationFiles(t *testing.T) {
	logger := logger.New()
	mockStorage := &storage.MockSqlStorage{}
	app := New(logger, mockStorage)

	migrationDir := "../migrations"
	migrationName := "create_users"

	app.Create(migrationName, migrationDir, "sql")

	upFile := fmt.Sprintf("%s/00001_%s_up.sql", migrationDir, migrationName)
	downFile := fmt.Sprintf("%s/00001_%s_down.sql", migrationDir, migrationName)
	assert.FileExists(t, upFile, "Expected Up migration file to be created")
	assert.FileExists(t, downFile, "Expected Down migration file to be created")

	os.Remove(upFile)
	os.Remove(downFile)
}

func TestUpMigration(t *testing.T) {
	logger := logger.New()
	mockStorage := &storage.MockSqlStorage{}
	app := New(logger, mockStorage)

	migrationDir := "../migrations"
	migrationName := "create_users"

	app.Create(migrationName, migrationDir, "sql")
	app.Up(migrationDir)

	migrations, _ := mockStorage.SelectMigrations(context.Background())
	assert.Equal(t, 1, len(migrations), "Expected one migration")
	assert.Equal(t, "create_users", migrations[0].GetName(), "Expected migration name to be 'create_users'")

	os.Remove(fmt.Sprintf("%s/00001_%s_up.sql", migrationDir, migrationName))
	os.Remove(fmt.Sprintf("%s/00001_%s_down.sql", migrationDir, migrationName))
}

func TestDownMigration(t *testing.T) {
	logger := logger.New()
	mockStorage := &storage.MockSqlStorage{}
	app := New(logger, mockStorage)

	migrationDir := "../migrations"
	migrationName := "create_users"

	app.Create(migrationName, migrationDir, "sql")
	app.Down(migrationDir)

	migrations, _ := mockStorage.SelectMigrations(context.Background())
	assert.Equal(t, 1, len(migrations), "Expected one migration")
	assert.Equal(t, "create_users", migrations[0].GetName(), "Expected migration name to be 'create_users'")

	os.Remove(fmt.Sprintf("%s/00001_%s_up.sql", migrationDir, migrationName))
	os.Remove(fmt.Sprintf("%s/00001_%s_down.sql", migrationDir, migrationName))
}
