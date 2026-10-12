package main

import (
	"automarket/handlers"
	"automarket/middleware"
	"automarket/repository"

	"github.com/gin-gonic/gin"
)

func main() {
	// 1. Inicializar la base de datos en memoria
	repository.InitDB()

	// 2. Crear el servidor web con Gin
	r := gin.Default()

	// 3. Aplicar el middleware de Accounting a TODAS las rutas
	r.Use(middleware.AccountingMiddleware())

	//Rutas

	// Registro autónomo del vendedor
	r.POST("/api/vendedor/registro", handlers.RegistrarVendedor)

	// Grupo de rutas para el vendedor (Debe estar autenticado)
	rutasVendedor := r.Group("/api/vendedor")
	{
		rutasVendedor.POST("/vehiculos", handlers.PublicarVehiculo)
		rutasVendedor.PUT("/vehiculos/:id/venta", handlers.ReportarVenta)
	}

	// Iniciar el servidor HTTP
	r.Run(":8080")
}
