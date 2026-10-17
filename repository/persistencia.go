package repository

import (
	"cmp"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"maps"
	"os"
	"path/filepath"
	"slices"

	"automarket/models"
)

// usuarioGuardado es la forma en que un usuario se escribe en el archivo.
//
// El modelo Usuario marca HashContrasena con json:"-" para que NUNCA salga
// en una respuesta. Pero en el archivo SÍ debe guardarse: sin el hash, nadie
// podría iniciar sesión después de reiniciar. Esta estructura "incrusta" el
// modelo (todos sus campos) y agrega el hash con un nombre propio.
type usuarioGuardado struct {
	models.Usuario
	HashContrasena string `json:"hash_contrasena"`
}

// estadoGuardado es el contenido completo del archivo de datos.
type estadoGuardado struct {
	Usuarios            []usuarioGuardado `json:"usuarios"`
	Vehiculos           []models.Vehiculo `json:"vehiculos"`
	SiguienteIDUsuario  int               `json:"siguiente_id_usuario"`
	SiguienteIDVehiculo int               `json:"siguiente_id_vehiculo"`
}

// AbrirArchivoDatos carga los usuarios y vehículos guardados en ejecuciones
// anteriores y activa el guardado automático después de cada cambio.
//
// Si el archivo todavía no existe (primera ejecución), se empieza vacío.
func (db *BaseDatosMemoria) AbrirArchivoDatos(ruta string) error {
	// Crear la carpeta si no existe (0o700: solo el dueño puede entrar).
	if err := os.MkdirAll(filepath.Dir(ruta), 0o700); err != nil {
		return err
	}

	db.mu.Lock()
	defer db.mu.Unlock()

	contenido, err := os.ReadFile(ruta)
	if errors.Is(err, fs.ErrNotExist) {
		db.rutaDatos = ruta
		return nil
	}
	if err != nil {
		return err
	}

	var estado estadoGuardado
	if err := json.Unmarshal(contenido, &estado); err != nil {
		return fmt.Errorf("el archivo de datos está dañado: %w", err)
	}

	for _, guardado := range estado.Usuarios {
		usuario := guardado.Usuario
		usuario.HashContrasena = guardado.HashContrasena
		db.usuarios[usuario.ID] = usuario
	}
	for _, vehiculo := range estado.Vehiculos {
		db.vehiculos[vehiculo.ID] = vehiculo
	}
	db.siguienteIDUsuario = estado.SiguienteIDUsuario
	db.siguienteIDVehiculo = estado.SiguienteIDVehiculo
	db.rutaDatos = ruta
	return nil
}

// guardarDatos escribe en el archivo el estado completo de usuarios y
// vehículos. Debe llamarse con el candado de escritura tomado.
// Si no se abrió un archivo (por ejemplo, en las pruebas), no hace nada.
func (db *BaseDatosMemoria) guardarDatos() error {
	if db.rutaDatos == "" {
		return nil
	}

	estado := estadoGuardado{
		Usuarios:            make([]usuarioGuardado, 0, len(db.usuarios)),
		Vehiculos:           make([]models.Vehiculo, 0, len(db.vehiculos)),
		SiguienteIDUsuario:  db.siguienteIDUsuario,
		SiguienteIDVehiculo: db.siguienteIDVehiculo,
	}
	for _, usuario := range db.usuarios {
		estado.Usuarios = append(estado.Usuarios, usuarioGuardado{
			Usuario:        usuario,
			HashContrasena: usuario.HashContrasena,
		})
	}
	for _, vehiculo := range db.vehiculos {
		estado.Vehiculos = append(estado.Vehiculos, vehiculo)
	}

	// Orden fijo: el archivo es fácil de leer y no cambia si los datos no cambian.
	slices.SortFunc(estado.Usuarios, func(a, b usuarioGuardado) int {
		return cmp.Compare(a.ID, b.ID)
	})
	ordenarPorID(estado.Vehiculos)

	// MarshalIndent escribe el JSON con sangría, para que una persona lo pueda leer.
	contenido, err := json.MarshalIndent(estado, "", "  ")
	if err != nil {
		return err
	}
	return escribirArchivoAtomico(db.rutaDatos, contenido)
}

// escribirArchivoAtomico reemplaza un archivo de una sola vez.
//
// Primero escribe en un archivo temporal y luego lo renombra. Si el programa
// se cae a mitad de la escritura, el archivo original queda intacto: nunca
// existe un archivo a medio escribir.
func escribirArchivoAtomico(ruta string, contenido []byte) error {
	temporal := ruta + ".tmp"
	// 0o600: solo el dueño puede leer y escribir. El archivo contiene los
	// hashes de las contraseñas, así que se trata como una base de datos.
	if err := os.WriteFile(temporal, contenido, 0o600); err != nil {
		return err
	}
	return os.Rename(temporal, ruta)
}

// respaldoMemoria es una copia del estado en memoria, tomada antes de un
// cambio para poder deshacerlo si no se logra guardar en disco.
type respaldoMemoria struct {
	usuarios            map[int]models.Usuario
	vehiculos           map[int]models.Vehiculo
	siguienteIDUsuario  int
	siguienteIDVehiculo int
}

// tomarRespaldo copia el estado actual. maps.Clone crea un mapa nuevo con
// los mismos elementos, así los cambios posteriores no afectan la copia.
func (db *BaseDatosMemoria) tomarRespaldo() respaldoMemoria {
	return respaldoMemoria{
		usuarios:            maps.Clone(db.usuarios),
		vehiculos:           maps.Clone(db.vehiculos),
		siguienteIDUsuario:  db.siguienteIDUsuario,
		siguienteIDVehiculo: db.siguienteIDVehiculo,
	}
}

// confirmar guarda los cambios en disco. Si falla, restaura el respaldo
// para que la memoria y el disco nunca queden distintos (todo o nada).
func (db *BaseDatosMemoria) confirmar(respaldo respaldoMemoria) error {
	if err := db.guardarDatos(); err != nil {
		db.usuarios = respaldo.usuarios
		db.vehiculos = respaldo.vehiculos
		db.siguienteIDUsuario = respaldo.siguienteIDUsuario
		db.siguienteIDVehiculo = respaldo.siguienteIDVehiculo
		return err
	}
	return nil
}
