package models

// Aquí realizar la parte que les corresponde
type LogEntry struct {
	Timestamp string
	Method    string
	Path      string
	ClientIP  string
}

type Vendedor struct {
	ID       int    `json:"id"`
	Nombre   string `json:"nombre"`
	Username string `json:"username"`
	Password string `json:"password"`
}

type Vehiculo struct {
	ID         int    `json:"id"`
	Marca      string `json:"marca"`
	Modelo     string `json:"modelo"`
	Precio     int    `json:"precio"`
	Estado     string `json:"estado"`
	VendedorID int    `json:"vendedor_id"`
}
