// Paquete handlers: contiene las funciones que atienden cada endpoint de la API.
// Cada handler lee la petición, llama a un servicio y traduce el resultado a HTTP.
package handlers

import (
	"net/http"

	"automarket/servicios"

	"github.com/gin-gonic/gin"
)

// respuestaVendedor contiene los datos del vendedor que se devuelven al
// cliente. No incluye la contraseña ni su hash.
type respuestaVendedor struct {
	ID            int    `json:"id"`
	Nombre        string `json:"nombre"`
	NombreUsuario string `json:"nombre_usuario"`
	Telefono      string `json:"telefono"`
}

// RegistrarVendedor permite que una persona se registre como vendedor (HU-VEN-01).
func RegistrarVendedor(c *gin.Context) {
	// Los campos que no existen en DatosRegistro (como "rol") se ignoran.
	var datos servicios.DatosRegistro
	if err := c.ShouldBindJSON(&datos); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "datos inválidos"})
		return
	}

	vendedor, err := servicios.RegistrarVendedor(datos)
	if err != nil {
		responderErrorServicio(c, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"mensaje": "vendedor registrado",
		"vendedor": respuestaVendedor{
			ID:            vendedor.ID,
			Nombre:        vendedor.Nombre,
			NombreUsuario: vendedor.NombreUsuario,
			Telefono:      vendedor.Telefono,
		},
	})
}

// PublicarVehiculo registra un vehículo en estado pendiente de aprobación,
// asociado al vendedor autenticado (HU-VEN-01).
func PublicarVehiculo(c *gin.Context) {
	vendedor, ok := leerUsuarioActual(c)
	if !ok {
		return
	}

	var datos servicios.DatosPublicacion
	if err := c.ShouldBindJSON(&datos); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "datos inválidos"})
		return
	}

	vehiculo, err := servicios.PublicarVehiculo(vendedor.ID, datos)
	if err != nil {
		responderErrorServicio(c, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"mensaje":  "vehículo registrado en estado pendiente de aprobación",
		"vehiculo": vehiculo,
	})
}

// ReportarVenta marca como vendido un vehículo publicado del vendedor
// autenticado (HU-VEN-01).
func ReportarVenta(c *gin.Context) {
	vendedor, ok := leerUsuarioActual(c)
	if !ok {
		return
	}

	id, ok := leerID(c)
	if !ok {
		return
	}

	vehiculo, err := servicios.ReportarVenta(vendedor.ID, id)
	if err != nil {
		responderErrorServicio(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"mensaje":  "venta reportada; el vehículo salió del catálogo",
		"vehiculo": vehiculo,
	})
}
