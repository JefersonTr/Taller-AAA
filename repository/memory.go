package repository

import (
	"sync"
	"automarket/models"
)

// Db simulada en memoria
type InMemoryDB struct {
	mu   sync.RWMutex
	logs []models.LogEntry
	// Aquí realizar los mapas de los vehiculos y usuarios 
}

var DB *InMemoryDB

func InitDB() {
	DB = &InMemoryDB{
		logs: make([]models.LogEntry, 0),
	}
}

func (db *InMemoryDB) SaveLog(entry models.LogEntry) {
	db.mu.Lock()
	defer db.mu.Unlock()
	db.logs = append(db.logs, entry)
}