// Paquete handlers: contiene las funciones que atienden cada endpoint de la API.
package handlers

import (
	"errors"
	"net/http"

	"automarket/auth"
	"automarket/models"
	"automarket/repository"

	"github.com/gin-gonic/gin"
)

// solicitudRegistro contiene los datos que el cliente envía para registrarse.
//
// Usamos una estructura distinta al modelo Usuario para que el cliente
// NO pueda enviar campos como "id" o "rol" (por ejemplo, registrarse
// como administrador). Solo se leen los campos que aparecen aquí.
// binding:"required" hace que Gin rechace la petición si falta el campo.
type solicitudRegistro struct {
	Nombre string `json:"nombre" binding:"required"`
	// excludes=: impide los dos puntos, porque Basic Auth los usa como separador
	NombreUsuario string `json:"nombre_usuario" binding:"required,excludes=:"`
	Telefono      string `json:"telefono" binding:"required"`
	Contrasena    string `json:"contrasena" binding:"required"`
}

// solicitudPublicacion contiene los datos que el vendedor envía al publicar.
// El estado y el vendedor NO vienen del cliente: los asigna el servidor.
type solicitudPublicacion struct {
	Marca  string `json:"marca" binding:"required"`
	Modelo string `json:"modelo" binding:"required"`
	Precio int    `json:"precio" binding:"required,gt=0"` // gt=0: mayor que cero
}

// respuestaVendedor contiene los datos del vendedor que se devuelven al cliente.
// No incluye la contraseña ni su hash.
type respuestaVendedor struct {
	ID            int    `json:"id"`
	Nombre        string `json:"nombre"`
	NombreUsuario string `json:"nombre_usuario"`
	Telefono      string `json:"telefono"`
}

// RegistrarVendedor permite que una persona se registre como vendedor (HU-VEN-01).
func RegistrarVendedor(c *gin.Context) {
	// 1. Leer y validar el JSON recibido.
	var solicitud solicitudRegistro
	if err := c.ShouldBindJSON(&solicitud); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "datos inválidos"})
		return
	}

	// 2. Convertir la contraseña en hash. La contraseña original no se guarda.
	hash, err := auth.GenerarHash(solicitud.Contrasena)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "no se pudo completar el registro"})
		return
	}

	// 3. Guardar el usuario. El rol lo asigna el servidor, nunca el cliente.
	vendedor, err := repository.DB.GuardarUsuario(models.Usuario{
		Nombre:         solicitud.Nombre,
		NombreUsuario:  solicitud.NombreUsuario,
		Telefono:       solicitud.Telefono,
		Rol:            models.RolVendedor,
		HashContrasena: hash,
	})
	if errors.Is(err, repository.ErrNombreUsuarioEnUso) {
		c.JSON(http.StatusConflict, gin.H{"error": "el nombre de usuario ya está registrado"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "no se pudo completar el registro"})
		return
	}

	// 4. Responder sin datos sensibles.
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

// PublicarVehiculo registra un vehículo en estado pendiente de aprobación (HU-VEN-01).
func PublicarVehiculo(c *gin.Context) {
	// 1. Leer y validar el JSON recibido.
	var solicitud solicitudPublicacion
	if err := c.ShouldBindJSON(&solicitud); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "datos inválidos"})
		return
	}

	// 2. Guardar el vehículo. El estado siempre inicia en pendiente.
	vehiculo := repository.DB.GuardarVehiculo(models.Vehiculo{
		Marca:  solicitud.Marca,
		Modelo: solicitud.Modelo,
		Precio: solicitud.Precio,
		Estado: models.EstadoPendiente,
		// TEMPORAL: cuando la autenticación esté conectada, aquí irá
		// el ID del vendedor autenticado en lugar del valor fijo 1.
		VendedorID: 1,
	})

	c.JSON(http.StatusCreated, gin.H{
		"mensaje":  "vehículo registrado en estado pendiente de aprobación",
		"vehiculo": vehiculo,
	})
}

// ReportarVenta cambiará el estado de un vehículo del vendedor a vendido.
// Mientras no esté implementada, responde 501 en lugar de simular un éxito.
func ReportarVenta(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{"error": "funcionalidad en desarrollo"})
}
