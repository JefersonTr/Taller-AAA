package handlers

import (
	"errors"
	"log"
	"net/http"
	"strconv"

	"automarket/middleware"
	"automarket/models"
	"automarket/servicios"

	"github.com/gin-gonic/gin"
)

// idDeRuta convierte el parámetro ":id" de la ruta en un número.
// No responde nada: solo indica si la conversión fue posible.
func idDeRuta(c *gin.Context) (int, bool) {
	id, err := strconv.Atoi(c.Param("id"))
	return id, err == nil
}

// leerID convierte el ":id" de la ruta y, si no es un número válido,
// responde 400 en JSON y devuelve false. La usan los handlers de la API.
func leerID(c *gin.Context) (int, bool) {
	id, ok := idDeRuta(c)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "identificador inválido"})
	}
	return id, ok
}

// leerUsuarioActual obtiene el usuario autenticado de la petición.
// Si no lo hay (una ruta configurada sin autenticación), responde 401 y
// devuelve false: ante la duda se niega la operación ("fallar cerrado").
func leerUsuarioActual(c *gin.Context) (models.Usuario, bool) {
	usuario, autenticado := middleware.UsuarioActual(c)
	if !autenticado {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "autenticación requerida"})
		return models.Usuario{}, false
	}
	return usuario, true
}

// traducirError devuelve el código HTTP y el mensaje que corresponden a un
// error de los servicios. La usan tanto la API (JSON) como las páginas (HTML),
// para que ambas respondan igual ante el mismo error.
func traducirError(err error) (int, string) {
	// En un switch sin expresión, "case a, b:" se cumple si a O b es verdadero.
	switch {
	case errors.Is(err, servicios.ErrDatosInvalidos):
		return http.StatusBadRequest, err.Error()
	case errors.Is(err, servicios.ErrNoEncontrado):
		return http.StatusNotFound, "recurso no encontrado"
	case errors.Is(err, servicios.ErrSinPermiso),
		errors.Is(err, servicios.ErrAdministradorProtegido):
		return http.StatusForbidden, err.Error()
	case errors.Is(err, servicios.ErrNombreUsuarioEnUso),
		errors.Is(err, servicios.ErrEstadoNoValido):
		return http.StatusConflict, err.Error()
	default:
		// Error inesperado: se registra en la consola del servidor para
		// investigarlo, pero al cliente no se le dan detalles internos.
		log.Printf("error interno: %v", err)
		return http.StatusInternalServerError, "error interno del servidor"
	}
}

// responderErrorServicio responde en JSON el error de un servicio.
func responderErrorServicio(c *gin.Context, err error) {
	codigo, mensaje := traducirError(err)
	c.JSON(codigo, gin.H{"error": mensaje})
}
