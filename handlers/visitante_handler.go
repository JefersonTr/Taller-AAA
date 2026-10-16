package handlers

import (
	"net/http"

	"automarket/servicios"

	"github.com/gin-gonic/gin"
)

// ListarCatalogo devuelve los vehículos publicados con el contacto de su
// vendedor (HU-VIS-01). Es una ruta pública: no requiere autenticación.
func ListarCatalogo(c *gin.Context) {
	catalogo, err := servicios.ListarCatalogo()
	if err != nil {
		responderErrorServicio(c, err)
		return
	}
	c.JSON(http.StatusOK, catalogo)
}

// ConsultarVehiculo devuelve un vehículo publicado con el contacto de su
// vendedor (HU-VIS-01). Es una ruta pública: no requiere autenticación.
func ConsultarVehiculo(c *gin.Context) {
	id, ok := leerID(c)
	if !ok {
		return
	}

	vehiculo, err := servicios.ConsultarVehiculo(id)
	if err != nil {
		responderErrorServicio(c, err)
		return
	}
	c.JSON(http.StatusOK, vehiculo)
}
