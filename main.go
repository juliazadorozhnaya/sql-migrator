package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"sql-migrator/app"
	"sql-migrator/logger"
)

var (
	ErrInvalidFlagNumber = errors.New("invalid flag number")

	path          string
	database      string
	migrationName string
)

func init() {
	flag.StringVar(&path, "path", "", "Path to migrations file")
	flag.StringVar(&database, "database", "", "Database connection string")
	flag.StringVar(&migrationName, "name", "", "Migration name")
}

func main() {
	flag.Parse()
	if len(flag.Args()) > 1 {
		fmt.Println(ErrInvalidFlagNumber)
		return
	}

	if path == "" {
		path = os.Getenv("path")
	}
	if database == "" {
		database = os.Getenv("database")
	}
	if migrationName == "" {
		migrationName = os.Getenv("name")
	}

	l := logger.New()
	application := app.New(l)

	switch flag.Arg(0) {
	case "create":
		application.Create(migrationName, path)
	case "up":
		application.Up(path, database)
	case "down":
		application.Down(path, database)
	case "redo":
		application.Redo(path, database)
	case "status":
		application.Status(database)
	case "dbversion":
		application.DbVersion(database)
	}
}
