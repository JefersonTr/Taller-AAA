package repository

import (
	"automarket/models"
	"sync"
)

// Db simulada en memoria
type InMemoryDB struct {
	mu   sync.RWMutex
	logs []models.LogEntry

	// Variables para parte 3
	vehiculos   map[int]models.Vehiculo
	vendedores  map[int]models.Vendedor
	nextVehicID int
	nextVendID  int
}

var DB *InMemoryDB

func InitDB() {
	DB = &InMemoryDB{
		logs: make([]models.LogEntry, 0),

		// Inicializacion de la parte 3
		vehiculos:   make(map[int]models.Vehiculo),
		vendedores:  make(map[int]models.Vendedor),
		nextVehicID: 1,
		nextVendID:  1,
	}
}

func (db *InMemoryDB) SaveLog(entry models.LogEntry) {
	db.mu.Lock()
	defer db.mu.Unlock()
	db.logs = append(db.logs, entry)
}

// Funciones para la parte 3
func (db *InMemoryDB) GuardarVendedor(v models.Vendedor) models.Vendedor {
	db.mu.Lock()
	defer db.mu.Unlock()
	v.ID = db.nextVendID
	db.vendedores[v.ID] = v
	db.nextVendID++
	return v
}

func (db *InMemoryDB) GuardarVehiculo(v models.Vehiculo) models.Vehiculo {
	db.mu.Lock()
	defer db.mu.Unlock()
	v.ID = db.nextVehicID
	db.vehiculos[v.ID] = v
	db.nextVehicID++
	return v
}
