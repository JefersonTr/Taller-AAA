// Paquete repository: guarda y consulta los datos de AutoMarket.
// Los datos viven en memoria y, si se abre un archivo con AbrirArchivoDatos,
// también se guardan en disco después de cada cambio.
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
// Los servicios y handlers los comparan con errors.Is.
var (
	ErrNombreUsuarioEnUso = errors.New("el nombre de usuario ya está registrado")
	ErrNoEncontrado       = errors.New("el recurso solicitado no existe")
)

// BaseDatosMemoria guarda la información en memoria, con guardado opcional en disco.
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

	usuarios  map[int]models.Usuario
	vehiculos map[int]models.Vehiculo

	// Contadores para asignar el siguiente ID disponible.
	siguienteIDUsuario  int
	siguienteIDVehiculo int

	// rutaDatos es el archivo donde se guardan usuarios y vehículos.
	// Si está vacía (por ejemplo, en las pruebas), los datos quedan solo en memoria.
	rutaDatos string
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

	respaldo := db.tomarRespaldo()
	usuario.ID = db.siguienteIDUsuario
	db.usuarios[usuario.ID] = usuario
	db.siguienteIDUsuario++

	if err := db.confirmar(respaldo); err != nil {
		return models.Usuario{}, err
	}
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
func (db *BaseDatosMemoria) BuscarUsuarioPorID(id int) (models.Usuario, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	usuario, existe := db.usuarios[id]
	if !existe {
		return models.Usuario{}, ErrNoEncontrado
	}
	return usuario, nil
}

// ListarUsuarios devuelve todos los usuarios registrados, ordenados por ID.
func (db *BaseDatosMemoria) ListarUsuarios() []models.Usuario {
	db.mu.RLock()
	defer db.mu.RUnlock()

	lista := make([]models.Usuario, 0, len(db.usuarios))
	for _, usuario := range db.usuarios {
		lista = append(lista, usuario)
	}
	slices.SortFunc(lista, func(a, b models.Usuario) int {
		return cmp.Compare(a.ID, b.ID)
	})
	return lista
}

// EliminarUsuario elimina al usuario y TODOS sus vehículos.
// Devuelve ErrNoEncontrado si el usuario no existe.
//
// Los vehículos se eliminan en la misma operación (dentro del mismo candado)
// para que nunca queden publicaciones de un vendedor que ya no existe.
func (db *BaseDatosMemoria) EliminarUsuario(id int) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	if _, existe := db.usuarios[id]; !existe {
		return ErrNoEncontrado
	}

	respaldo := db.tomarRespaldo()
	// En Go es seguro borrar elementos de un mapa mientras se recorre.
	for idVehiculo, vehiculo := range db.vehiculos {
		if vehiculo.VendedorID == id {
			delete(db.vehiculos, idVehiculo)
		}
	}
	delete(db.usuarios, id)

	return db.confirmar(respaldo)
}

// ---------------- Vehículos ----------------

// GuardarVehiculo asigna un ID al vehículo y lo guarda.
func (db *BaseDatosMemoria) GuardarVehiculo(vehiculo models.Vehiculo) (models.Vehiculo, error) {
	db.mu.Lock()
	defer db.mu.Unlock()

	respaldo := db.tomarRespaldo()
	vehiculo.ID = db.siguienteIDVehiculo
	db.vehiculos[vehiculo.ID] = vehiculo
	db.siguienteIDVehiculo++

	if err := db.confirmar(respaldo); err != nil {
		return models.Vehiculo{}, err
	}
	return vehiculo, nil
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
// Las reglas sobre qué cambios están permitidos se validan en los
// servicios; el repositorio solo guarda.
func (db *BaseDatosMemoria) ActualizarEstadoVehiculo(id int, estado models.EstadoVehiculo) (models.Vehiculo, error) {
	db.mu.Lock()
	defer db.mu.Unlock()

	// En un mapa de structs se obtiene una COPIA: hay que volver a guardarla.
	vehiculo, existe := db.vehiculos[id]
	if !existe {
		return models.Vehiculo{}, ErrNoEncontrado
	}

	respaldo := db.tomarRespaldo()
	vehiculo.Estado = estado
	db.vehiculos[id] = vehiculo

	if err := db.confirmar(respaldo); err != nil {
		return models.Vehiculo{}, err
	}
	return vehiculo, nil
}

// EliminarVehiculo elimina un vehículo.
// Devuelve ErrNoEncontrado si el vehículo no existe.
func (db *BaseDatosMemoria) EliminarVehiculo(id int) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	if _, existe := db.vehiculos[id]; !existe {
		return ErrNoEncontrado
	}

	respaldo := db.tomarRespaldo()
	delete(db.vehiculos, id)
	return db.confirmar(respaldo)
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
	ordenarPorID(lista)
	return lista
}

// ListarVehiculosPorVendedor devuelve todos los vehículos de un vendedor,
// en cualquier estado, ordenados por ID.
func (db *BaseDatosMemoria) ListarVehiculosPorVendedor(vendedorID int) []models.Vehiculo {
	db.mu.RLock()
	defer db.mu.RUnlock()

	lista := make([]models.Vehiculo, 0)
	for _, vehiculo := range db.vehiculos {
		if vehiculo.VendedorID == vendedorID {
			lista = append(lista, vehiculo)
		}
	}
	ordenarPorID(lista)
	return lista
}

// ordenarPorID ordena una lista de vehículos por ID, de menor a mayor.
// Los mapas en Go no tienen un orden fijo, así que sin esto las listas
// cambiarían de orden en cada consulta.
func ordenarPorID(lista []models.Vehiculo) {
	slices.SortFunc(lista, func(a, b models.Vehiculo) int {
		return cmp.Compare(a.ID, b.ID)
	})
}
