package app

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path"
	"regexp"
	"strconv"
	"strings"

	"github.com/juliazadorozhnaya/sql-migrator/logger"
	"github.com/juliazadorozhnaya/sql-migrator/processes"
	"github.com/juliazadorozhnaya/sql-migrator/storage"
)

type App interface {
	Create(name, path string, migrationType string)
	Up(path string)
	Down(path string)
	Redo(path string)
	Status()
	DbVersion()
}

type Application struct {
	logger     logger.Logger
	sqlStorage storage.SqlStorage
}

var (
	ErrInvalidMigrationName = errors.New("invalid migration name")

	regGetVersion         = regexp.MustCompile(`^\d+`)
	regGetUpMigration     = regexp.MustCompile(`^.+_up\.sql$`)
	regGetDownMigration   = regexp.MustCompile(`^.+_down\.sql$`)
	regGetUpGoMigration   = regexp.MustCompile(`^.+_up\.go$`)
	regGetDownGoMigration = regexp.MustCompile(`^.+_down\.go$`)
)

func New(logger logger.Logger, sqlStorage storage.SqlStorage) *Application {
	return &Application{
		logger:     logger,
		sqlStorage: sqlStorage,
	}
}

func (app *Application) Create(name, filePath, migrationType string) {
	files, err := os.ReadDir(filePath)
	if err != nil {
		app.logger.Fatal("Failed to read directory: ", err)
		return
	}

	lastVersion := getLastVersion(files, app.logger)
	if lastVersion < 0 {
		return
	}

	lastVersion++

	if err := createMigrationFiles(filePath, lastVersion, name, app.logger, migrationType); err != nil {
		app.logger.Fatal("Failed to create migration files: ", err)
	}
}

func (app *Application) Up(filePath string) {
	app.runMigrations(filePath, func(migrator *processes.Migrator, ctx context.Context) error {
		return migrator.Up(ctx)
	})
}

func (app *Application) Down(filePath string) {
	app.runMigrations(filePath, func(migrator *processes.Migrator, ctx context.Context) error {
		return migrator.Down(ctx)
	})
}

func (app *Application) Redo(filePath string) {
	app.runMigrations(filePath, func(migrator *processes.Migrator, ctx context.Context) error {
		return migrator.Redo(ctx)
	})
}

func (app *Application) Status() {
	app.runSingleCommand(func(migrator *processes.Migrator, ctx context.Context) error {
		return migrator.Status(ctx)
	})
}

// DbVersion выводит текущую версию базы данных
func (app *Application) DbVersion() {
	app.runSingleCommand(func(migrator *processes.Migrator, ctx context.Context) error {
		return migrator.DbVersion(ctx)
	})
}

func (app *Application) runMigrations(filePath string, migrationFunc func(*processes.Migrator, context.Context) error) {
	migrator := processes.New(app.sqlStorage, app.logger)
	migrations, err := getMigrations(filePath)
	if err != nil {
		app.logger.Fatal("Failed to get migrations: ", err)
		return
	}

	for _, migration := range migrations {
		migrator.Create(migration.Name, migration.Up, migration.Down, migration.UpGo, migration.DownGo)
	}

	ctx := context.Background()
	if err := migrator.Connect(ctx); err != nil {
		app.logger.Fatal("Failed to connect to database: ", err)
		return
	}
	defer migrator.Close(ctx)

	if err := migrationFunc(migrator, ctx); err != nil {
		app.logger.Error("Migration failed: ", err)
	}
}

func (app *Application) runSingleCommand(commandFunc func(*processes.Migrator, context.Context) error) {
	migrator := processes.New(app.sqlStorage, app.logger)
	ctx := context.Background()
	if err := migrator.Connect(ctx); err != nil {
		app.logger.Fatal("Failed to connect to database: ", err)
		return
	}
	defer migrator.Close(ctx)

	if err := commandFunc(migrator, ctx); err != nil {
		app.logger.Error("Command failed: ", err)
	}
}

func getLastVersion(files []os.DirEntry, logger logger.Logger) int {
	lastVersion := 0

	for _, file := range files {
		strVersion := regGetVersion.FindString(file.Name())

		if strVersion != "" {
			version, err := strconv.Atoi(strVersion)
			if err != nil {
				logger.Error("Failed to parse version: ", err)
				return -1
			}

			if version > lastVersion {
				lastVersion = version
			}
		}
	}

	return lastVersion
}

func createMigrationFiles(filePath string, version int, name string, logger logger.Logger, migrationType string) error {
	switch migrationType {
	case "sql":
		upFile := path.Join(filePath, fmt.Sprintf("%05d_%s_up.sql", version, name))
		err := os.WriteFile(upFile, nil, 0644)
		if err != nil {
			return err
		}
		logger.Info(upFile + " created")

		downFile := path.Join(filePath, fmt.Sprintf("%05d_%s_down.sql", version, name))
		err = os.WriteFile(downFile, nil, 0644)
		if err != nil {
			return err
		}
		logger.Info(downFile + " created")
	case "go":
		upFile := path.Join(filePath, fmt.Sprintf("%05d_%s_up.go", version, name))
		upContent := `package main

import (
	"context"
	"github.com/juliazadorozhnaya/sql-migrator/storage"
)

func Up(ctx context.Context) error {
	db, ok := ctx.Value("db").(*storage.SqlStorage)
	if !ok {
		return fmt.Errorf("could not get database connection from context")
	}

	sql := "
		CREATE TABLE IF NOT EXISTS users (
		id SERIAL PRIMARY KEY,
		username VARCHAR(255) NOT NULL,
		email VARCHAR(255) NOT NULL UNIQUE,
		created_at TIMESTAMP NOT NULL DEFAULT NOW()
		);"

	if err := db.Migrate(ctx, sql); err != nil {
		return fmt.Errorf("could not execute migration: %v", err)
	}

	fmt.Println("Migration Up applied: users table created")
	return nil
}
`
		err := os.WriteFile(upFile, []byte(upContent), 0644)
		if err != nil {
			return err
		}
		logger.Info(upFile + " created")

		downFile := path.Join(filePath, fmt.Sprintf("%05d_%s_down.go", version, name))
		downContent := `package main

import (
	"context"
	"github.com/juliazadorozhnaya/sql-migrator/storage"
)

func Down(ctx context.Context) error {
	db, ok := ctx.Value("db").(*storage.SqlStorage)
	if !ok {
		return fmt.Errorf("could not get database connection from context")
	}

	sql := "DROP TABLE IF EXISTS users;""

	if err := db.Migrate(ctx, sql); err != nil {
		return fmt.Errorf("could not execute migration: %v", err)
	}

	fmt.Println("Migration Down applied: users table dropped")
	return nil
}
`
		err = os.WriteFile(downFile, []byte(downContent), 0644)
		if err != nil {
			return err
		}
		logger.Info(downFile + " created")
	default:
		return errors.New("unsupported migration type")
	}
	return nil
}

func getMigrations(filePath string) (map[int]*storage.Migration, error) {
	files, err := os.ReadDir(filePath)
	if err != nil {
		return nil, err
	}

	migrations := make(map[int]*storage.Migration)

	for _, file := range files {
		strVersion := regGetVersion.FindString(file.Name())

		if strVersion != "" {
			version, err := strconv.Atoi(strVersion)
			if err != nil {
				return nil, err
			}

			parts := strings.Split(file.Name(), "_")
			if len(parts) != 3 {
				return nil, ErrInvalidMigrationName
			}

			sql, err := os.ReadFile(path.Join(filePath, file.Name()))
			if err != nil {
				return nil, err
			}

			if regGetUpMigration.MatchString(file.Name()) {
				if _, ok := migrations[version]; ok {
					migrations[version].Up = string(sql)
				} else {
					migrations[version] = &storage.Migration{
						Version: version,
						Name:    parts[1],
						Up:      string(sql),
					}
				}
			} else if regGetDownMigration.MatchString(file.Name()) {
				if _, ok := migrations[version]; ok {
					migrations[version].Down = string(sql)
				} else {
					migrations[version] = &storage.Migration{
						Version: version,
						Name:    parts[1],
						Down:    string(sql),
					}
				}
			} else if regGetUpGoMigration.MatchString(file.Name()) {
				if _, ok := migrations[version]; ok {
					migrations[version].UpGo = func(ctx context.Context) error {
						return runGoMigration(filePath, file.Name())
					}
				} else {
					migrations[version] = &storage.Migration{
						Version: version,
						Name:    parts[1],
						UpGo: func(ctx context.Context) error {
							return runGoMigration(filePath, file.Name())
						},
					}
				}
			} else if regGetDownGoMigration.MatchString(file.Name()) {
				if _, ok := migrations[version]; ok {
					migrations[version].DownGo = func(ctx context.Context) error {
						return runGoMigration(filePath, file.Name())
					}
				} else {
					migrations[version] = &storage.Migration{
						Version: version,
						Name:    parts[1],
						DownGo: func(ctx context.Context) error {
							return runGoMigration(filePath, file.Name())
						},
					}
				}
			} else {
				return nil, ErrInvalidMigrationName
			}
		}
	}

	return migrations, nil
}

func runGoMigration(filePath, fileName string) error {
	cmd := exec.Command("go", "run", path.Join(filePath, fileName))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
