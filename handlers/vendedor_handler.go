package handlers

import (
	"automarket/models"
	"automarket/repository"
	"net/http"

	"github.com/gin-gonic/gin"
)

// Permite a un vendedor registrarse
func RegistrarVendedor(c *gin.Context) {
	var nuevoVendedor models.Vendedor
	if err := c.ShouldBindJSON(&nuevoVendedor); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Datos inválidos"})
		return
	}

	// Llama a la función GuardarVendedor de memory.go
	vendedorGuardado := repository.DB.GuardarVendedor(nuevoVendedor)
	c.JSON(http.StatusCreated, gin.H{"mensaje": "Vendedor registrado", "vendedor": vendedorGuardado})
}

// Registra un vehículo obligando a que su estado sea "pendiente"
func PublicarVehiculo(c *gin.Context) {
	var nuevoVehiculo models.Vehiculo
	if err := c.ShouldBindJSON(&nuevoVehiculo); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Datos inválidos"})
		return
	}

	// Fuerza el estado a pendiente según el criterio de aceptación HU-VEN-01
	nuevoVehiculo.Estado = "pendiente"

	// Simulado mientras se implementa la PARTE 2
	//
	nuevoVehiculo.VendedorID = 1

	// Llama a la función GuardarVehiculo de memory.go
	vehiculoGuardado := repository.DB.GuardarVehiculo(nuevoVehiculo)
	c.JSON(http.StatusCreated, gin.H{"mensaje": "Vehículo registrado en estado pendiente", "vehiculo": vehiculoGuardado})
}

// Simula el cambio de estado de un vehículo a vendido
func ReportarVenta(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"mensaje": "Venta reportada exitosamente"})
}
