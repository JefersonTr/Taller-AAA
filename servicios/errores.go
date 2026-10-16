// Paquete servicios: contiene las reglas de negocio de AutoMarket.
//
// Las usan tanto los handlers de la API JSON como los de las páginas web,
// para que cada regla se escriba una sola vez. Este paquete no conoce HTTP:
// devuelve datos o errores de negocio, y cada handler los traduce a su manera.
package servicios

import (
	"errors"

	"automarket/repository"
)

// Errores de negocio que pueden devolver los servicios.
var (
	// ErrNoEncontrado: el recurso solicitado no existe.
	// Es el MISMO error del repositorio, re-exportado aquí para que los
	// handlers dependan solo de los servicios y no del repositorio.
	ErrNoEncontrado = repository.ErrNoEncontrado

	// ErrEstadoNoValido: la operación no aplica al estado actual del
	// vehículo (por ejemplo, autorizar una publicación que ya está publicada).
	ErrEstadoNoValido = errors.New("la operación no es válida para el estado actual del vehículo")

	// ErrAdministradorProtegido: se intentó eliminar a un administrador.
	ErrAdministradorProtegido = errors.New("no se puede eliminar a un administrador")
)
