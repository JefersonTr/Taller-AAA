// Paquete models: define las entidades (estructuras de datos) de AutoMarket.
// No contiene lógica; solo describe cómo son los datos del sistema.
package models

// Rol indica qué tipo de usuario autenticado es una persona.
// El visitante no aparece aquí porque no se registra ni se autentica.
type Rol string

const (
	RolVendedor      Rol = "vendedor"
	RolAdministrador Rol = "administrador"
)

// EstadoVehiculo indica en qué etapa está una publicación.
// Usar constantes evita errores de tipeo: si alguien escribe
// EstadoPendinte, el compilador lo detecta; con el texto "pendinte", no.
type EstadoVehiculo string

const (
	EstadoPendiente EstadoVehiculo = "pendiente" // Enviada por el vendedor, espera aprobación
	EstadoPublicada EstadoVehiculo = "publicada" // Aprobada por el administrador, visible en el catálogo
	EstadoVendido   EstadoVehiculo = "vendido"   // Reportada como vendida, sale del catálogo
)

// Usuario representa a una persona registrada: vendedor o administrador.
type Usuario struct {
	ID            int    `json:"id"`
	Nombre        string `json:"nombre"`
	NombreUsuario string `json:"nombre_usuario"`
	Telefono      string `json:"telefono"` // Dato de contacto que ve el visitante
	Rol           Rol    `json:"rol"`

	// HashContrasena guarda "sal$hash" generado por auth.GenerarHash.
	// La etiqueta json:"-" impide que este campo se envíe en cualquier respuesta.
	HashContrasena string `json:"-"`
}

// Vehiculo representa una publicación de un vehículo en venta.
type Vehiculo struct {
	ID         int            `json:"id"`
	Marca      string         `json:"marca"`
	Modelo     string         `json:"modelo"`
	Precio     int            `json:"precio"`
	Estado     EstadoVehiculo `json:"estado"`
	VendedorID int            `json:"vendedor_id"` // ID del Usuario que publicó el vehículo
}

// RegistroAuditoria guarda la información de una operación ejecutada
// sobre la API (la "A" de Accounting en el marco AAA).
type RegistroAuditoria struct {
	FechaHora    string `json:"fecha_hora"`    // Momento en que llegó la petición (UTC)
	Metodo       string `json:"metodo"`        // GET, POST, PUT...
	Ruta         string `json:"ruta"`          // Endpoint solicitado
	IPCliente    string `json:"ip_cliente"`    // Dirección IP de quien hizo la petición
	Usuario      string `json:"usuario"`       // Quién la hizo: nombre de usuario o "anonimo"
	CodigoEstado int    `json:"codigo_estado"` // Resultado: 200, 401, 403, 404...

}
