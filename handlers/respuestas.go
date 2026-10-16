package handlers

import (
	"errors"
	"log"
	"net/http"
	"strconv"

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

// responderErrorServicio traduce un error de los servicios al código HTTP
// que le corresponde. Así todos los handlers responden igual ante el mismo error.
func responderErrorServicio(c *gin.Context, err error) {
	// Un switch sin expresión funciona como una cadena de if / else if:
	// se ejecuta el primer caso cuya condición sea verdadera.
	switch {
	case errors.Is(err, servicios.ErrNoEncontrado):
		c.JSON(http.StatusNotFound, gin.H{"error": "recurso no encontrado"})
	case errors.Is(err, servicios.ErrEstadoNoValido):
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
	case errors.Is(err, servicios.ErrAdministradorProtegido):
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
	default:
		// Error inesperado: se registra en la consola del servidor para
		// poder investigarlo, pero al cliente no se le dan detalles internos.
		log.Printf("error interno: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "error interno del servidor"})
	}
}
