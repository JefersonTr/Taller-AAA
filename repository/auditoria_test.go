package repository

import (
	"errors"
	"slices"
	"testing"

	"automarket/models"
)

func TestCadenaAuditoria(t *testing.T) {
	// Preparación: tres operaciones registradas.
	InicializarBD()
	for _, ruta := range []string{"/api/vehiculos", "/api/vehiculos/1", "/api/vendedor/registro"} {
		DB.GuardarRegistro(models.RegistroAuditoria{
			FechaHora: "2026-10-15T15:00:00Z", Metodo: "GET", Ruta: ruta,
			IPCliente: "::1", Usuario: "anonimo", CodigoEstado: 200,
		})
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
	// slices.Clone hace una copia para no dañar los datos originales.
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
