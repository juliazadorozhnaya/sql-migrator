package app

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path"
	"regexp"
	"sql-migrator/logger"
	"sql-migrator/migration"
	"sql-migrator/storage"
	"strconv"
	"strings"
)

// App интерфейс для приложения миграций
type App interface {
	Create(name, path string)
	Up(path, connString string)
	Down(path, connString string)
	Redo(path, connString string)
	Status(connString string)
	DbVersion(connString string)
}

// Application структура приложения, реализующая интерфейс App
type Application struct {
	logger logger.Logger
}

var (
	ErrInvalidMigrationName = errors.New("invalid migration name")

	regGetVersion       = regexp.MustCompile(`^\d+`)
	regGetUpMigration   = regexp.MustCompile(`^.+_up\.sql$`)
	regGetDownMigration = regexp.MustCompile(`^.+_down\.sql$`)
)

// New создает новый экземпляр приложения
func New(logger logger.Logger) App {
	return &Application{
		logger: logger,
	}
}

func (app *Application) Create(name, filePath string) {
	files, err := os.ReadDir(filePath)
	if err != nil {
		app.logger.Error(err.Error())
		return
	}

	lastVersion := 0

	for _, file := range files {
		strVersion := regGetVersion.FindString(file.Name())

		if strVersion != "" {
			version, err := strconv.Atoi(strVersion)
			if err != nil {
				app.logger.Error(err.Error())
				return
			}

			if version > lastVersion {
				lastVersion = version
			}
		}
	}

	lastVersion++

	upFile := path.Join(filePath, fmt.Sprintf("%05d_%s_up.sql", lastVersion, name))
	err = os.WriteFile(upFile, nil, 0644)
	if err != nil {
		app.logger.Error(err.Error())
		return
	}
	app.logger.Info(upFile + " created")

	downFile := path.Join(filePath, fmt.Sprintf("%05d_%s_down.sql", lastVersion, name))
	err = os.WriteFile(downFile, nil, 0644)
	if err != nil {
		app.logger.Error(err.Error())
		return
	}
	app.logger.Info(downFile + " created")
}

func (app *Application) Up(filePath, connString string) {
	migrator := migration.New(connString, app.logger)
	migrations, err := getMigrations(filePath)
	if err != nil {
		app.logger.Error(err.Error())
		return
	}

	for i := 1; i <= len(migrations); i++ {
		if _, ok := migrations[i]; ok {
			migrator.Create(migrations[i].Name, migrations[i].Up, migrations[i].Down)
		}
	}

	ctx := context.Background()
	if err = migrator.Connect(ctx); err != nil {
		app.logger.Error(err.Error())
		return
	}
	defer migrator.Close(ctx)

	if err = migrator.Up(ctx); err != nil {
		app.logger.Error(err.Error())
		return
	}
}

func (app *Application) Down(filePath, connString string) {
	migrator := migration.New(connString, app.logger)
	ctx := context.Background()
	migrations, err := getMigrations(filePath)
	if err != nil {
		app.logger.Error(err.Error())
		return
	}

	for i := 1; i <= len(migrations); i++ {
		if _, ok := migrations[i]; ok {
			migrator.Create(migrations[i].Name, migrations[i].Up, migrations[i].Down)
		}
	}

	if err = migrator.Connect(ctx); err != nil {
		app.logger.Error(err.Error())
		return
	}
	defer migrator.Close(ctx)

	if err = migrator.Down(ctx); err != nil {
		app.logger.Error(err.Error())
		return
	}
}

func (app *Application) Redo(filePath, connString string) {
	migrator := migration.New(connString, app.logger)
	ctx := context.Background()
	migrations, err := getMigrations(filePath)
	if err != nil {
		app.logger.Error(err.Error())
		return
	}

	for i := 1; i <= len(migrations); i++ {
		if _, ok := migrations[i]; ok {
			migrator.Create(migrations[i].Name, migrations[i].Up, migrations[i].Down)
		}
	}

	if err = migrator.Connect(ctx); err != nil {
		app.logger.Error(err.Error())
		return
	}
	defer migrator.Close(ctx)

	if err = migrator.Redo(ctx); err != nil {
		app.logger.Error(err.Error())
		return
	}
}

func (app *Application) Status(connString string) {
	migrator := migration.New(connString, app.logger)
	ctx := context.Background()

	if err := migrator.Connect(ctx); err != nil {
		app.logger.Error(err.Error())
		return
	}
	defer migrator.Close(ctx)

	if err := migrator.Status(ctx); err != nil {
		app.logger.Error(err.Error())
		return
	}
}

func (app *Application) DbVersion(connString string) {
	migrator := migration.New(connString, app.logger)
	ctx := context.Background()

	if err := migrator.Connect(ctx); err != nil {
		app.logger.Error(err.Error())
		return
	}
	defer migrator.Close(ctx)

	if err := migrator.DbVersion(ctx); err != nil {
		app.logger.Error(err.Error())
		return
	}
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
			} else {
				return nil, ErrInvalidMigrationName
			}
		}
	}

	return migrations, nil
}
