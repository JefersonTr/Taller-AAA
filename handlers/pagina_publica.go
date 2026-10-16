package handlers

import (
	"net/http"

	"automarket/servicios"

	"github.com/gin-gonic/gin"
)

// PaginaCatalogo muestra los vehículos publicados (HU-VIS-01).
// Es pública: no requiere autenticación.
func PaginaCatalogo(c *gin.Context) {
	catalogo, err := servicios.ListarCatalogo()
	if err != nil {
		mostrarErrorServicio(c, err)
		return
	}

	c.HTML(http.StatusOK, "catalogo.html", gin.H{
		"Titulo":    "Catálogo",
		"Vehiculos": catalogo,
	})
}

// PaginaVehiculo muestra un vehículo publicado con el contacto de su
// vendedor (HU-VIS-01). Es pública: no requiere autenticación.
func PaginaVehiculo(c *gin.Context) {
	// En las páginas, un ID que no es número se trata como "no encontrado":
	// para quien navega no hay diferencia entre los dos casos.
	id, ok := idDeRuta(c)
	if !ok {
		mostrarError(c, http.StatusNotFound, "vehículo no encontrado")
		return
	}

	vehiculo, err := servicios.ConsultarVehiculo(id)
	if err != nil {
		mostrarErrorServicio(c, err)
		return
	}

	c.HTML(http.StatusOK, "vehiculo.html", gin.H{
		"Titulo":   vehiculo.Marca + " " + vehiculo.Modelo,
		"Vehiculo": vehiculo,
	})
}
