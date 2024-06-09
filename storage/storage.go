package storage

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v4/pgxpool"
)

// Storage описывает интерфейс для работы с хранилищем миграций
type Storage interface {
	SelectMigrations(context.Context) ([]IMigration, error)
	SelectLastMigrationByStatus(context.Context, string) (IMigration, error)
	Connect(context.Context) error
	Close() error

	InsertMigration(context.Context, IMigration) error
	Migrate(context.Context, string) error
	DeleteMigrations(context.Context) error
}

// sqlStorage представляет собой реализацию интерфейса Storage
type sqlStorage struct {
	connString string
	pool       *pgxpool.Pool
}

// Константы статусов миграций
const (
	StatusProcess      = "process"
	StatusSuccess      = "success"
	StatusError        = "error"
	StatusCancellation = "cancellation"
	StatusCancel       = "cancel"
)

// Определение ошибок
var (
	ErrUnexpectedStatus  = errors.New("unexpected status")
	ErrMigrationNotFound = errors.New("migration not found")
)

// New создает новый экземпляр sqlStorage
func New(connString string) Storage {
	return &sqlStorage{
		connString: connString,
	}
}

// Connect подключается к базе данных и создает таблицу для миграций, если она не существует
func (storage *sqlStorage) Connect(ctx context.Context) error {
	pool, err := pgxpool.Connect(ctx, storage.connString)
	if err != nil {
		return err
	}

	sql := `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			Version INTEGER,
			Name CHARACTER VARYING(100),
			Status CHARACTER VARYING(20),
			StatusChangeTime TIMESTAMP
		);`

	_, err = pool.Exec(ctx, sql)
	if err != nil {
		return err
	}

	storage.pool = pool
	return nil
}

// Close закрывает пул соединений с базой данных
func (storage *sqlStorage) Close() error {
	if storage.pool != nil {
		storage.pool.Close()
	}
	return nil
}

// DeleteMigrations удаляет все записи из таблицы schema_migrations
func (storage *sqlStorage) DeleteMigrations(ctx context.Context) error {
	_, err := storage.pool.Exec(ctx, "TRUNCATE schema_migrations;")
	return err
}

// SelectMigrations выбирает все миграции из таблицы schema_migrations
func (storage *sqlStorage) SelectMigrations(ctx context.Context) ([]IMigration, error) {
	sql := `SELECT Name, Status, Version, StatusChangeTime FROM schema_migrations ORDER BY Version DESC;`

	rows, err := storage.pool.Query(ctx, sql)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var migrations []IMigration
	for rows.Next() {
		var (
			name             string
			version          int
			status           string
			statusChangeTime time.Time
		)

		err = rows.Scan(&name, &status, &version, &statusChangeTime)
		if err != nil {
			return nil, err
		}

		migrations = append(migrations, NewMigration(name, status, version, statusChangeTime))
	}

	if len(migrations) == 0 {
		return nil, ErrMigrationNotFound
	}

	return migrations, nil
}

// SelectLastMigrationByStatus выбирает последнюю миграцию с указанным статусом из таблицы schema_migrations
func (storage *sqlStorage) SelectLastMigrationByStatus(ctx context.Context, status string) (IMigration, error) {
	switch status {
	case StatusSuccess, StatusError, StatusProcess, StatusCancellation, StatusCancel:
	default:
		return nil, ErrUnexpectedStatus
	}

	sql := `SELECT Name, Status, Version, StatusChangeTime FROM schema_migrations WHERE Status = $1 ORDER BY Version DESC LIMIT 1;`

	rows, err := storage.pool.Query(ctx, sql, status)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	if rows.Next() {
		var (
			name             string
			version          int
			status           string
			statusChangeTime time.Time
		)

		err = rows.Scan(&name, &status, &version, &statusChangeTime)
		if err != nil {
			return nil, err
		}

		return NewMigration(name, status, version, statusChangeTime), nil
	}

	return nil, ErrMigrationNotFound
}

// InsertMigration вставляет или обновляет запись миграции в таблице schema_migrations
func (storage *sqlStorage) InsertMigration(ctx context.Context, migration IMigration) error {
	sql := `
		DO $$ 
		BEGIN
			IF EXISTS (SELECT 1 FROM schema_migrations WHERE Version = $1 AND Name = $2) THEN
				UPDATE schema_migrations 
				SET Status = $3, StatusChangeTime = $4 
				WHERE Version = $1 AND Name = $2;
			ELSE
				INSERT INTO schema_migrations (Version, Name, Status, StatusChangeTime)
				VALUES ($1, $2, $3, $4);
			END IF;
		END $$;`

	_, err := storage.pool.Exec(ctx, sql, migration.GetVersion(), migration.GetName(), migration.GetStatus(), migration.GetStatusChangeTime())
	return err
}

// Migrate выполняет SQL запрос для миграции
func (storage *sqlStorage) Migrate(ctx context.Context, sql string) error {
	_, err := storage.pool.Exec(ctx, sql)
	return err
}
