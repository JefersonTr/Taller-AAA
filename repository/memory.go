// Paquete repository: guarda y consulta los datos de AutoMarket.
// Esta versión guarda todo en memoria, por lo que los datos
// se pierden cuando se detiene el servidor.
package repository

import (
	"cmp"
	"errors"
	"os"
	"slices"
	"sync"

	"automarket/models"
)

// Errores que pueden devolver las funciones de este paquete.
// Los handlers los comparan con errors.Is para decidir qué código HTTP responder.
var (
	ErrNombreUsuarioEnUso = errors.New("el nombre de usuario ya está registrado")
	ErrNoEncontrado       = errors.New("el recurso solicitado no existe")
)

// BaseDatosMemoria simula una base de datos guardando la información en memoria.
type BaseDatosMemoria struct {
	// mu es un candado que protege los datos cuando llegan varias peticiones
	// al mismo tiempo. Gin atiende cada petición en paralelo, y si dos
	// escriben un mapa a la vez, el programa se cierra con error.
	//   - Lock/Unlock: para escribir (solo uno a la vez).
	//   - RLock/RUnlock: para leer (varios a la vez).
	mu sync.RWMutex

	registros []models.RegistroAuditoria
	// archivoAuditoria es el archivo donde se agrega cada registro.
	// Si es nil (por ejemplo, en las pruebas), la auditoría queda solo en memoria.
	archivoAuditoria *os.File
	usuarios         map[int]models.Usuario
	vehiculos        map[int]models.Vehiculo

	// Contadores para asignar el siguiente ID disponible.
	siguienteIDUsuario  int
	siguienteIDVehiculo int
}

// DB es la única instancia de la base de datos que usa todo el proyecto.
var DB *BaseDatosMemoria

// InicializarBD crea la base de datos vacía. Se llama una sola vez al arrancar.
func InicializarBD() {
	DB = &BaseDatosMemoria{
		registros:           make([]models.RegistroAuditoria, 0),
		usuarios:            make(map[int]models.Usuario),
		vehiculos:           make(map[int]models.Vehiculo),
		siguienteIDUsuario:  1,
		siguienteIDVehiculo: 1,
	}
}

// ---------------- Usuarios ----------------

// GuardarUsuario asigna un ID al usuario y lo guarda.
// Devuelve ErrNombreUsuarioEnUso si el nombre de usuario ya existe.
func (db *BaseDatosMemoria) GuardarUsuario(usuario models.Usuario) (models.Usuario, error) {
	db.mu.Lock()
	defer db.mu.Unlock()

	// Verificar y guardar dentro del mismo candado evita que dos registros
	// simultáneos con el mismo nombre de usuario pasen ambos la verificación.
	for _, existente := range db.usuarios {
		if existente.NombreUsuario == usuario.NombreUsuario {
			return models.Usuario{}, ErrNombreUsuarioEnUso
		}
	}

	usuario.ID = db.siguienteIDUsuario
	db.usuarios[usuario.ID] = usuario
	db.siguienteIDUsuario++
	return usuario, nil
}

// BuscarUsuarioPorNombreUsuario devuelve el usuario con ese nombre de usuario,
// o ErrNoEncontrado si no existe. La usa la autenticación.
func (db *BaseDatosMemoria) BuscarUsuarioPorNombreUsuario(nombreUsuario string) (models.Usuario, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	for _, usuario := range db.usuarios {
		if usuario.NombreUsuario == nombreUsuario {
			return usuario, nil
		}
	}
	return models.Usuario{}, ErrNoEncontrado
}

// BuscarUsuarioPorID devuelve el usuario con ese ID, o ErrNoEncontrado.
// La usa el catálogo para obtener el contacto del vendedor.
func (db *BaseDatosMemoria) BuscarUsuarioPorID(id int) (models.Usuario, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	usuario, existe := db.usuarios[id]
	if !existe {
		return models.Usuario{}, ErrNoEncontrado
	}
	return usuario, nil
}

// ---------------- Vehículos ----------------

// GuardarVehiculo asigna un ID al vehículo y lo guarda.
func (db *BaseDatosMemoria) GuardarVehiculo(vehiculo models.Vehiculo) models.Vehiculo {
	db.mu.Lock()
	defer db.mu.Unlock()

	vehiculo.ID = db.siguienteIDVehiculo
	db.vehiculos[vehiculo.ID] = vehiculo
	db.siguienteIDVehiculo++
	return vehiculo
}

// BuscarVehiculoPorID devuelve el vehículo con ese ID, o ErrNoEncontrado.
func (db *BaseDatosMemoria) BuscarVehiculoPorID(id int) (models.Vehiculo, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	vehiculo, existe := db.vehiculos[id]
	if !existe {
		return models.Vehiculo{}, ErrNoEncontrado
	}
	return vehiculo, nil
}

// ActualizarEstadoVehiculo cambia el estado de un vehículo existente y
// devuelve el vehículo actualizado, o ErrNoEncontrado si no existe.
//
// La usan el vendedor (reportar venta: publicada → vendido) y el
// administrador (autorizar publicación: pendiente → publicada).
// Las reglas sobre qué cambios están permitidos se validan en los
// handlers; el repositorio solo guarda.
func (db *BaseDatosMemoria) ActualizarEstadoVehiculo(id int, estado models.EstadoVehiculo) (models.Vehiculo, error) {
	db.mu.Lock()
	defer db.mu.Unlock()

	// Paso 1: obtener una COPIA del vehículo guardado en el mapa.
	vehiculo, existe := db.vehiculos[id]
	if !existe {
		return models.Vehiculo{}, ErrNoEncontrado
	}

	// Paso 2: cambiar el estado en la copia y volver a guardarla.
	// Sin esta segunda línea, el cambio se perdería.
	vehiculo.Estado = estado
	db.vehiculos[id] = vehiculo
	return vehiculo, nil
}

// ListarVehiculosPorEstado devuelve los vehículos en el estado indicado,
// ordenados por ID para que el resultado sea siempre el mismo.
func (db *BaseDatosMemoria) ListarVehiculosPorEstado(estado models.EstadoVehiculo) []models.Vehiculo {
	db.mu.RLock()
	defer db.mu.RUnlock()

	// make(..., 0) crea una lista vacía (no nula): en JSON se verá como []
	// en lugar de null cuando no haya vehículos.
	lista := make([]models.Vehiculo, 0)
	for _, vehiculo := range db.vehiculos {
		if vehiculo.Estado == estado {
			lista = append(lista, vehiculo)
		}
	}

	// Los mapas en Go no tienen orden fijo; ordenamos por ID de menor a mayor.
	slices.SortFunc(lista, func(a, b models.Vehiculo) int {
		return cmp.Compare(a.ID, b.ID)
	})
	return lista
}
