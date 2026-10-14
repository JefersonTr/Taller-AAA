package handlers

import (
	"errors"
	"net/http"
	"strconv"

	"automarket/models"
	"automarket/repository"

	"github.com/gin-gonic/gin"
)

// contactoVendedor contiene los únicos datos del vendedor que puede ver
// un visitante. No incluye nombre de usuario, rol ni contraseña.
type contactoVendedor struct {
	Nombre   string `json:"nombre"`
	Telefono string `json:"telefono"`
}

// vehiculoPublico es la información de un vehículo que ve un visitante.
// No incluye el estado ni el ID interno del vendedor.
type vehiculoPublico struct {
	ID       int              `json:"id"`
	Marca    string           `json:"marca"`
	Modelo   string           `json:"modelo"`
	Precio   int              `json:"precio"`
	Vendedor contactoVendedor `json:"vendedor"`
}

// ListarCatalogo devuelve los vehículos publicados con el contacto de su
// vendedor (HU-VIS-01). Es una ruta pública: no requiere autenticación.
func ListarCatalogo(c *gin.Context) {
	vehiculos := repository.DB.ListarVehiculosPorEstado(models.EstadoPublicada)

	// make con capacidad len(vehiculos): reserva espacio de una vez.
	catalogo := make([]vehiculoPublico, 0, len(vehiculos))
	for _, vehiculo := range vehiculos {
		publico, err := armarVehiculoPublico(vehiculo)
		if errors.Is(err, repository.ErrNoEncontrado) {
			// El vendedor ya no existe: el vehículo no se muestra,
			// porque un visitante no podría contactar a nadie.
			continue
		}
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "error interno del servidor"})
			return
		}
		catalogo = append(catalogo, publico)
	}

	c.JSON(http.StatusOK, catalogo)
}

// ConsultarVehiculo devuelve un vehículo publicado con el contacto de su
// vendedor (HU-VIS-01). Es una ruta pública: no requiere autenticación.
func ConsultarVehiculo(c *gin.Context) {
	// 1. Convertir el ":id" de la ruta (texto) a número.
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "identificador inválido"})
		return
	}

	// 2. Buscar el vehículo.
	vehiculo, err := repository.DB.BuscarVehiculoPorID(id)
	if err != nil && !errors.Is(err, repository.ErrNoEncontrado) {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "error interno del servidor"})
		return
	}

	// 3. Un vehículo inexistente y uno no publicado reciben la MISMA respuesta.
	//    Así el visitante no puede descubrir publicaciones pendientes o vendidas.
	if err != nil || vehiculo.Estado != models.EstadoPublicada {
		responderVehiculoNoEncontrado(c)
		return
	}

	// 4. Agregar el contacto del vendedor.
	publico, err := armarVehiculoPublico(vehiculo)
	if errors.Is(err, repository.ErrNoEncontrado) {
		responderVehiculoNoEncontrado(c)
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "error interno del servidor"})
		return
	}

	c.JSON(http.StatusOK, publico)
}

// armarVehiculoPublico combina un vehículo con el contacto de su vendedor.
// La usan los dos endpoints para no repetir código.
func armarVehiculoPublico(vehiculo models.Vehiculo) (vehiculoPublico, error) {
	vendedor, err := repository.DB.BuscarUsuarioPorID(vehiculo.VendedorID)
	if err != nil {
		return vehiculoPublico{}, err
	}

	return vehiculoPublico{
		ID:     vehiculo.ID,
		Marca:  vehiculo.Marca,
		Modelo: vehiculo.Modelo,
		Precio: vehiculo.Precio,
		Vendedor: contactoVendedor{
			Nombre:   vendedor.Nombre,
			Telefono: vendedor.Telefono,
		},
	}, nil
}

// responderVehiculoNoEncontrado envía la respuesta 404 del catálogo.
func responderVehiculoNoEncontrado(c *gin.Context) {
	c.JSON(http.StatusNotFound, gin.H{"error": "vehículo no encontrado"})
}
