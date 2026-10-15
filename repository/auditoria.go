package repository

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"

	"automarket/models"
)

// ErrCadenaAlterada indica que el registro de auditoría fue modificado
// o que se eliminaron registros intermedios.
var ErrCadenaAlterada = errors.New("el registro de auditoría fue alterado")

// calcularHashRegistro calcula el SHA-256 de un registro de auditoría,
// incluido el hash del registro anterior (pero no su propio campo Hash).
//
// %q escribe cada texto entre comillas y escapa los caracteres especiales.
// Así ningún contenido puede "correrse" de un campo a otro: por ejemplo,
// Ruta="/a" y Usuario="b c" nunca producen lo mismo que Ruta="/a b" y Usuario="c".
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
