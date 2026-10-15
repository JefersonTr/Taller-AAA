package repository

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"automarket/models"
)

// ErrCadenaAlterada indica que el registro de auditoría fue modificado,
// que se eliminaron registros intermedios o que el archivo está dañado.
var ErrCadenaAlterada = errors.New("el registro de auditoría fue alterado")

// GuardarRegistro agrega una operación al registro de auditoría,
// encadenándola con el registro anterior mediante su hash.
func (db *BaseDatosMemoria) GuardarRegistro(registro models.RegistroAuditoria) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	// El encadenamiento ocurre DENTRO del candado. Si dos peticiones
	// llegaran a la vez fuera del candado, ambas podrían tomar el mismo
	// "registro anterior" y la cadena se bifurcaría.
	if len(db.registros) > 0 {
		registro.HashAnterior = db.registros[len(db.registros)-1].Hash
	}
	registro.Hash = calcularHashRegistro(registro)

	// Primero el archivo, después la memoria. Si el disco falla, el registro
	// no se agrega a ninguno de los dos y la cadena del archivo sigue intacta.
	if db.archivoAuditoria != nil {
		if err := escribirRegistro(db.archivoAuditoria, registro); err != nil {
			return err
		}
	}

	db.registros = append(db.registros, registro)
	return nil
}

// AbrirArchivoAuditoria carga los registros guardados en ejecuciones
// anteriores, verifica que la cadena esté intacta y deja el archivo
// abierto para agregar los nuevos registros al final.
//
// Si la cadena fue alterada, devuelve ErrCadenaAlterada y no abre el archivo.
func (db *BaseDatosMemoria) AbrirArchivoAuditoria(ruta string) error {
	// Paso 1: crear la carpeta si no existe.
	// 0o700 = solo el dueño puede leer, escribir y entrar (en Linux/macOS).
	if err := os.MkdirAll(filepath.Dir(ruta), 0o700); err != nil {
		return err
	}

	// Paso 2: leer lo guardado en ejecuciones anteriores.
	registros, err := leerRegistros(ruta)
	if err != nil {
		return err
	}

	// Paso 3: verificar la cadena ANTES de aceptar registros nuevos.
	if err := verificarCadena(registros); err != nil {
		return err
	}

	// Paso 4: abrir el archivo para AGREGAR al final (O_APPEND), crearlo si
	// no existe (O_CREATE) y solo escribir (O_WRONLY). Nunca se sobrescribe.
	archivo, err := os.OpenFile(ruta, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}

	db.mu.Lock()
	defer db.mu.Unlock()
	db.registros = registros
	db.archivoAuditoria = archivo
	return nil
}

// leerRegistros lee el archivo de auditoría, un registro JSON por línea.
// Si el archivo todavía no existe (primera ejecución), devuelve una lista vacía.
func leerRegistros(ruta string) ([]models.RegistroAuditoria, error) {
	archivo, err := os.Open(ruta)
	if errors.Is(err, fs.ErrNotExist) {
		return make([]models.RegistroAuditoria, 0), nil
	}
	if err != nil {
		return nil, err
	}
	defer archivo.Close()

	registros := make([]models.RegistroAuditoria, 0)
	lector := bufio.NewScanner(archivo) // Lee el archivo línea por línea
	for lector.Scan() {
		var registro models.RegistroAuditoria
		if err := json.Unmarshal(lector.Bytes(), &registro); err != nil {
			// Una línea que no es JSON válido también es una alteración.
			return nil, fmt.Errorf("%w: la línea %d no se puede leer", ErrCadenaAlterada, len(registros)+1)
		}
		registros = append(registros, registro)
	}
	return registros, lector.Err()
}

// escribirRegistro agrega un registro al final del archivo como una línea JSON.
func escribirRegistro(archivo *os.File, registro models.RegistroAuditoria) error {
	linea, err := json.Marshal(registro)
	if err != nil {
		return err
	}
	_, err = archivo.Write(append(linea, '\n'))
	return err
}

// calcularHashRegistro calcula el SHA-256 de un registro de auditoría,
// incluido el hash del registro anterior (pero no su propio campo Hash).
//
// %q escribe cada texto entre comillas y escapa los caracteres especiales.
// Así ningún contenido puede "correrse" de un campo a otro.
//
// IMPORTANTE: si se agrega un campo a RegistroAuditoria, hay que agregarlo
// aquí; si no, ese campo podría alterarse sin que la cadena lo detecte.
func calcularHashRegistro(registro models.RegistroAuditoria) string {
	contenido := fmt.Sprintf("%q %q %q %q %q %d %q",
		registro.FechaHora,
		registro.Metodo,
		registro.Ruta,
		registro.IPCliente,
		registro.Usuario,
		registro.CodigoEstado,
		registro.HashAnterior,
	)
	suma := sha256.Sum256([]byte(contenido))
	return hex.EncodeToString(suma[:])
}

// verificarCadena recorre los registros en orden y comprueba que cada uno:
//  1. apunte al hash del registro anterior, y
//  2. conserve el hash que le corresponde a su contenido.
//
// Devuelve ErrCadenaAlterada indicando el primer registro con problemas.
func verificarCadena(registros []models.RegistroAuditoria) error {
	hashAnterior := ""
	for i, registro := range registros {
		if registro.HashAnterior != hashAnterior {
			return fmt.Errorf("%w: el registro %d no apunta al anterior", ErrCadenaAlterada, i+1)
		}
		if registro.Hash != calcularHashRegistro(registro) {
			return fmt.Errorf("%w: el contenido del registro %d fue modificado", ErrCadenaAlterada, i+1)
		}
		hashAnterior = registro.Hash
	}
	return nil
}
