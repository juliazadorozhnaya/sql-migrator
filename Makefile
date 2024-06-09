BIN := ./bin/sql-migrator
MAIN := ./cmd

test:
	go test -race -count 100 ./pkg/...
	go test -race -count 100 ./internal/...

integration-tests:
	cd deployments && docker-compose -f docker-compose.test.yaml build && docker-compose -f docker-compose.test.yaml up

install-lint-deps:
	(which golangci-lint > /dev/null) || curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(shell go env GOPATH)/bin v1.50.0

lint: install-lint-deps
	golangci-lint run ./...

postgres:
	docker run --name postgresdb --env POSTGRES_PASSWORD="1234512345" --publish "5436:5432" --detach --rm postgres

build:
	go build -o ${BIN} ${MAIN}

run: build
	${BIN} --path migrations --name "init" create

up: build
	${BIN} --path migrations --database "postgresql://postgres:1234512345@localhost:5436/postgres?sslmode=disable" up

down: build
	${BIN} --path migrations --database "postgresql://postgres:1234512345@localhost:5436/postgres?sslmode=disable" down

redo: build
	${BIN} --path migrations --database "postgresql://postgres:1234512345@localhost:5436/postgres?sslmode=disable" redo

status: build
	${BIN} --database "postgresql://postgres:1234512345@localhost:5436/postgres?sslmode=disable" status

dbversion: build
	${BIN} --database "postgresql://postgres:1234512345@localhost:5436/postgres?sslmode=disable" dbversion

.PHONY: test integration-tests install-lint-deps lint postgres build \
		run up down status dbversion
