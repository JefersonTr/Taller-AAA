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

// leerID convierte el parámetro ":id" de la ruta en un número.
// Si no es un número válido, responde 400 y devuelve false; en ese caso
// el handler que la llamó debe terminar con return.
func leerID(c *gin.Context) (int, bool) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "identificador inválido"})
		return 0, false
	}
	return id, true
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

// responderErrorServicio traduce un error de los servicios al código HTTP
// que le corresponde. Así todos los handlers responden igual ante el mismo error.
func responderErrorServicio(c *gin.Context, err error) {
	// En un switch sin expresión, "case a, b:" se cumple si a O b es verdadero.
	switch {
	case errors.Is(err, servicios.ErrDatosInvalidos):
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	case errors.Is(err, servicios.ErrNoEncontrado):
		c.JSON(http.StatusNotFound, gin.H{"error": "recurso no encontrado"})
	case errors.Is(err, servicios.ErrSinPermiso),
		errors.Is(err, servicios.ErrAdministradorProtegido):
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
	case errors.Is(err, servicios.ErrNombreUsuarioEnUso),
		errors.Is(err, servicios.ErrEstadoNoValido):
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
	default:
		// Error inesperado: se registra en la consola del servidor para
		// investigarlo, pero al cliente no se le dan detalles internos.
		log.Printf("error interno: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "error interno del servidor"})
	}
}
