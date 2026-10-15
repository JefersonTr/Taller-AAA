package repository

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"automarket/models"
)

// registroDePrueba crea un registro de auditoría con datos fijos.
func registroDePrueba(ruta, usuario string) models.RegistroAuditoria {
	return models.RegistroAuditoria{
		FechaHora: "2026-10-15T15:00:00Z", Metodo: "GET", Ruta: ruta,
		IPCliente: "::1", Usuario: usuario, CodigoEstado: 200,
	}
}

func TestCadenaAuditoria(t *testing.T) {
	// Preparación: tres operaciones registradas (solo en memoria).
	InicializarBD()
	for _, ruta := range []string{"/api/vehiculos", "/api/vehiculos/1", "/api/vendedor/registro"} {
		if err := DB.GuardarRegistro(registroDePrueba(ruta, "anonimo")); err != nil {
			t.Fatal(err)
		}
	}
	registros := DB.registros

	// Caso 1: la cadena recién creada es válida.
	if err := verificarCadena(registros); err != nil {
		t.Fatal("una cadena sin alterar debe ser válida:", err)
	}

	// Caso 2: cada registro apunta al anterior; el primero, a "".
	if registros[0].HashAnterior != "" || registros[1].HashAnterior != registros[0].Hash {
		t.Fatal("los registros deben quedar encadenados")
	}

	// Caso 3: modificar un registro intermedio se detecta.
	modificados := slices.Clone(registros)
	modificados[1].Usuario = "admin"
	if err := verificarCadena(modificados); !errors.Is(err, ErrCadenaAlterada) {
		t.Fatal("debería detectar un registro modificado")
	}

	// Caso 4: eliminar un registro intermedio se detecta.
	sinIntermedio := []models.RegistroAuditoria{registros[0], registros[2]}
	if err := verificarCadena(sinIntermedio); !errors.Is(err, ErrCadenaAlterada) {
		t.Fatal("debería detectar un registro eliminado")
	}
}

func TestPersistenciaAuditoria(t *testing.T) {
	// t.TempDir() crea una carpeta temporal que Go borra al terminar la prueba.
	archivo := filepath.Join(t.TempDir(), "auditoria.jsonl")

	// Primera "ejecución" del servidor: registra dos operaciones.
	InicializarBD()
	if err := DB.AbrirArchivoAuditoria(archivo); err != nil {
		t.Fatal(err)
	}
	for _, ruta := range []string{"/api/vehiculos", "/api/vehiculos/1"} {
		if err := DB.GuardarRegistro(registroDePrueba(ruta, "ana")); err != nil {
			t.Fatal(err)
		}
	}
	DB.archivoAuditoria.Close() // Simula apagar el servidor

	// Segunda "ejecución": recupera lo anterior y continúa la cadena.
	InicializarBD()
	if err := DB.AbrirArchivoAuditoria(archivo); err != nil {
		t.Fatal("un archivo sin alterar debe cargarse:", err)
	}
	if len(DB.registros) != 2 {
		t.Fatalf("se esperaban 2 registros recuperados, hay %d", len(DB.registros))
	}
	if err := DB.GuardarRegistro(registroDePrueba("/api/vendedor/registro", "anonimo")); err != nil {
		t.Fatal(err)
	}
	DB.archivoAuditoria.Close()

	// El archivo debe tener los 3 registros, todos encadenados.
	registros, err := leerRegistros(archivo)
	if err != nil || len(registros) != 3 || verificarCadena(registros) != nil {
		t.Fatal("el archivo debe contener 3 registros con la cadena intacta")
	}

	// Ataque: alguien edita el archivo para cambiar quién hizo una operación.
	contenido, err := os.ReadFile(archivo)
	if err != nil {
		t.Fatal(err)
	}
	alterado := strings.Replace(string(contenido), `"usuario":"ana"`, `"usuario":"luis"`, 1)
	if alterado == string(contenido) {
		t.Fatal("la prueba no logró alterar el archivo")
	}
	if err := os.WriteFile(archivo, []byte(alterado), 0o600); err != nil {
		t.Fatal(err)
	}

	// Tercera "ejecución": debe detectar la alteración y negarse a cargar.
	InicializarBD()
	if err := DB.AbrirArchivoAuditoria(archivo); !errors.Is(err, ErrCadenaAlterada) {
		t.Fatal("debería detectar que el archivo fue alterado")
	}
}
