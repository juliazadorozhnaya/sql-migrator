package storage

import (
	"time"
)

// IMigration представляет интерфейс для работы с миграцией.
type IMigration interface {
	GetName() string
	GetStatus() string
	GetVersion() int
	GetStatusChangeTime() time.Time

	SetName(name string)
	SetStatus(status string)
	SetVersion(version int)
	SetStatusChangeTime(statusChangeTime time.Time)
}

// Migration реализует интерфейс IMigration и содержит информацию о миграции.
type Migration struct {
	Name             string
	Version          int
	Status           string
	StatusChangeTime time.Time
	Up               string
	Down             string
}

// NewMigration создает новый объект migration.
func NewMigration(name, status string, version int, statusChangeTime time.Time) IMigration {
	return &Migration{
		Name:             name,
		Status:           status,
		Version:          version,
		StatusChangeTime: statusChangeTime,
	}
}

// GetName возвращает имя миграции.
func (m *Migration) GetName() string {
	return m.Name
}

// GetStatus возвращает статус миграции.
func (m *Migration) GetStatus() string {
	return m.Status
}

// GetVersion возвращает версию миграции.
func (m *Migration) GetVersion() int {
	return m.Version
}

// GetStatusChangeTime возвращает время последнего изменения статуса миграции.
func (m *Migration) GetStatusChangeTime() time.Time {
	return m.StatusChangeTime
}

// SetName устанавливает имя миграции.
func (m *Migration) SetName(name string) {
	m.Name = name
}

// SetStatus устанавливает статус миграции.
func (m *Migration) SetStatus(status string) {
	m.Status = status
}

// SetVersion устанавливает версию миграции.
func (m *Migration) SetVersion(version int) {
	m.Version = version
}

// SetStatusChangeTime устанавливает время последнего изменения статуса миграции.
func (m *Migration) SetStatusChangeTime(statusChangeTime time.Time) {
	m.StatusChangeTime = statusChangeTime
}
