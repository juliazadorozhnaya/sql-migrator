package processes

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/juliazadorozhnaya/sql-migrator/logger"
	"github.com/juliazadorozhnaya/sql-migrator/storage"
)

type IMigration interface {
	Connect(context.Context) error
	Close(context.Context) error
	Create(name, up, down string, upGo, downGo func(ctx context.Context) error)
	Up(context.Context) error
	Down(context.Context) error
	Redo(context.Context) error
	Status(context.Context) error
	DbVersion(context.Context) error
}

type Migrator struct {
	logger     logger.Logger
	storage    storage.SqlStorage
	migrations []storage.Migration
}

var (
	ErrMigrationUp                = errors.New("error processes up")
	ErrMigrationDown              = errors.New("error processes down")
	ErrMigrationRedo              = errors.New("error processes redo")
	ErrGetStatus                  = errors.New("error db status")
	ErrGetVersion                 = errors.New("error db version")
	ErrUnexpectedMigrationVersion = errors.New("unexpected processes version")
)

func New(connString storage.SqlStorage, logger logger.Logger) *Migrator {
	return &Migrator{
		storage:    connString,
		logger:     logger,
		migrations: make([]storage.Migration, 0),
	}
}

func (m *Migrator) Connect(ctx context.Context) error {
	m.logger.Info("Connecting to database")

	err := m.storage.Connect(ctx)
	if err != nil {
		m.logger.Error("Error in Connect: %v", err)
		return err
	}

	m.logger.Info("Connected to database")
	return nil
}

func (m *Migrator) Close(ctx context.Context) error {
	m.logger.Info("Closing database connection")

	err := m.storage.Close()
	if err != nil {
		m.logger.Error("Error in Close: %v", err)
		return err
	}

	m.logger.Info("Database connection closed")
	return nil
}

func (m *Migrator) Create(name, up, down string, upGo, downGo func(ctx context.Context) error) {
	m.logger.Info("Creating migration: %s", name)
	m.migrations = append(m.migrations, storage.Migration{
		Version: len(m.migrations) + 1,
		Name:    name,
		Up:      up,
		Down:    down,
		UpGo:    upGo,
		DownGo:  downGo,
	})
	m.logger.Info("Migration %s created", name)
}

func (m *Migrator) Up(ctx context.Context) error {
	m.logger.Info("Starting migrations")

	if err := m.storage.Lock(ctx); err != nil {
		m.logger.Error("Error in Up: %v", err)
		return err
	}
	defer m.storage.Unlock(ctx)

	lastVersion := 0
	lastMigration, err := m.storage.SelectLastMigrationByStatus(ctx, storage.StatusSuccess)
	if err == nil {
		lastVersion = lastMigration.GetVersion()
	} else if !errors.Is(err, storage.ErrMigrationNotFound) {
		m.logger.Error("Error in Up: %v", err)
		return err
	}

	if lastMigration != nil && lastMigration.GetVersion()-1 > len(m.migrations) {
		m.logger.Error("Error in Up: %v", ErrUnexpectedMigrationVersion)
		return ErrUnexpectedMigrationVersion
	}

	for i := lastVersion; i < len(m.migrations); i++ {
		err = m.upMigration(ctx, &m.migrations[i], m.migrations[i].Up, m.migrations[i].UpGo)
		if err != nil {
			m.logger.Error("Error in Up: %v", err)
			return ErrMigrationUp
		}
	}

	m.logger.Info("Migrations completed")
	return nil
}

func (m *Migrator) Down(ctx context.Context) error {
	m.logger.Info("Starting rollback")

	if err := m.storage.Lock(ctx); err != nil {
		m.logger.Error("Error in Down: %v", err)
		return err
	}
	defer m.storage.Unlock(ctx)

	lastMigration, err := m.storage.SelectLastMigrationByStatus(ctx, storage.StatusSuccess)
	if err != nil {
		m.logger.Error("Error in Down: %v", err)
		return err
	}

	if lastMigration != nil && lastMigration.GetVersion()-1 > len(m.migrations) {
		m.logger.Error("Error in Down: %v", ErrUnexpectedMigrationVersion)
		return ErrUnexpectedMigrationVersion
	}

	downMigrationIndex := lastMigration.GetVersion() - 1
	err = m.downMigration(ctx, &m.migrations[downMigrationIndex], m.migrations[downMigrationIndex].Down, m.migrations[downMigrationIndex].DownGo)
	if err != nil {
		m.logger.Error("Error in Down: %v", err)
		return ErrMigrationDown
	}

	m.logger.Info("Rollback completed")
	return nil
}

func (m *Migrator) upMigration(ctx context.Context, migration storage.IMigration, sql string, upGo func(ctx context.Context) error) error {
	migration.SetStatus(storage.StatusProcess)
	migration.SetStatusChangeTime(time.Now())

	if err := m.storage.InsertMigration(ctx, migration); err != nil {
		m.logger.Error("Error in upMigration: %v", err)
		return err
	}

	if upGo != nil {
		if err := upGo(ctx); err != nil {
			migration.SetStatus(storage.StatusError)
			migration.SetStatusChangeTime(time.Now())
			m.storage.InsertMigration(ctx, migration)

			m.logger.Error("Error in upMigration: %v", err)
			return err
		}
	} else if sql != "" {
		if err := m.storage.Migrate(ctx, sql); err != nil {
			migration.SetStatus(storage.StatusError)
			migration.SetStatusChangeTime(time.Now())
			m.storage.InsertMigration(ctx, migration)

			m.logger.Error("Error in upMigration: %v", err)
			return err
		}
	}

	migration.SetStatus(storage.StatusSuccess)
	migration.SetStatusChangeTime(time.Now())
	if err := m.storage.InsertMigration(ctx, migration); err != nil {
		m.logger.Error("Error in upMigration: %v", err)
		return err
	}

	m.logger.Info("Migration %s to version %d applied successfully", migration.GetName(), migration.GetVersion())
	return nil
}

func (m *Migrator) downMigration(ctx context.Context, migration storage.IMigration, sql string, downGo func(ctx context.Context) error) error {
	migration.SetStatus(storage.StatusCancellation)
	migration.SetStatusChangeTime(time.Now())

	if err := m.storage.InsertMigration(ctx, migration); err != nil {
		m.logger.Error("Error in downMigration: %v", err)
		return err
	}

	if downGo != nil {
		if err := downGo(ctx); err != nil {
			migration.SetStatus(storage.StatusError)
			migration.SetStatusChangeTime(time.Now())
			m.storage.InsertMigration(ctx, migration)

			m.logger.Error("Error in downMigration: %v", err)
			return err
		}
	} else if sql != "" {
		if err := m.storage.Migrate(ctx, sql); err != nil {
			migration.SetStatus(storage.StatusError)
			migration.SetStatusChangeTime(time.Now())
			m.storage.InsertMigration(ctx, migration)

			m.logger.Error("Error in downMigration: %v", err)
			return err
		}
	}

	migration.SetStatus(storage.StatusCancel)
	migration.SetStatusChangeTime(time.Now())
	if err := m.storage.InsertMigration(ctx, migration); err != nil {
		m.logger.Error("Error in downMigration: %v", err)
		return err
	}

	m.logger.Info("Rollback of migration %s to version %d applied successfully", migration.GetName(), migration.GetVersion())
	return nil
}

func (m *Migrator) Redo(ctx context.Context) error {
	m.logger.Info("Starting redo process")

	err := m.Down(ctx)
	if err != nil {
		m.logger.Error("Error in Redo: %v", err)
		return err
	}

	lastVersion := 0
	lastMigration, err := m.storage.SelectLastMigrationByStatus(ctx, storage.StatusSuccess)
	if err == nil {
		lastVersion = lastMigration.GetVersion()
	} else if !errors.Is(err, storage.ErrMigrationNotFound) {
		m.logger.Error("Error in Redo: %v", err)
		return err
	}

	if lastMigration != nil && lastMigration.GetVersion()-1 > len(m.migrations) {
		m.logger.Error("Error in Redo: %v", ErrUnexpectedMigrationVersion)
		return ErrUnexpectedMigrationVersion
	}

	err = m.upMigration(ctx, &m.migrations[lastVersion], m.migrations[lastVersion].Up, m.migrations[lastVersion].UpGo)
	if err != nil {
		m.logger.Error("Error in Redo: %v", err)
		return ErrMigrationRedo
	}

	m.logger.Info("Redo process completed")
	return nil
}

func (m *Migrator) Status(ctx context.Context) error {
	migrations, err := m.storage.SelectMigrations(ctx)
	if err != nil {
		m.logger.Error("Error in Status: %v", err)
		return ErrGetStatus
	}

	m.logger.Info("._____________________._____________________._____________________.")
	m.logger.Info("| %-19s | %-19s | %-19s |", "Название", "Статус", "Время")

	for _, migr := range migrations {
		formatMigration := fmt.Sprintf("| %-19s | %-19s | %s |",
			migr.GetName(), migr.GetStatus(), migr.GetStatusChangeTime().Format("2006-01-02 15:04:05"))

		m.logger.Info(formatMigration)
	}

	m.logger.Info("|_____________________|_____________________|_____________________|")
	return nil
}

func (m *Migrator) DbVersion(ctx context.Context) error {
	lastVersion := 0

	lastMigration, err := m.storage.SelectLastMigrationByStatus(ctx, storage.StatusSuccess)
	if err == nil {
		lastVersion = lastMigration.GetVersion()
	} else if !errors.Is(err, storage.ErrMigrationNotFound) {
		m.logger.Error("Error in DbVersion: %v", err)
		return ErrGetVersion
	}

	m.logger.Info("Version: %d", lastVersion)
	return nil
}
