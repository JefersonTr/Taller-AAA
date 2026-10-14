// Punto de entrada de AutoMarket: configura el servidor y registra las rutas.
package main

import (
	"log"

	"automarket/handlers"
	"automarket/middleware"
	"automarket/repository"

	"github.com/gin-gonic/gin"
)

func main() {
	// 1. Crear la base de datos en memoria.
	repository.InicializarBD()

	// 2. Crear el servidor SIN middlewares predeterminados.
	//    gin.Default() agrega su propio logger, que duplicaría el Accounting.
	r := gin.New()

	// 3. No confiar en ningún proxy. Así c.ClientIP() usa la IP real de la
	//    conexión y no un header X-Forwarded-For que el cliente puede falsificar.
	if err := r.SetTrustedProxies(nil); err != nil {
		log.Fatalf("no se pudo configurar los proxies de confianza: %v", err)
	}

	// 4. Middlewares globales. El orden importa:
	//    - Accounting primero, para registrar TODAS las operaciones.
	//    - Recovery después: si un handler falla de forma inesperada,
	//      Recovery responde 500 y Accounting alcanza a registrarlo.
	r.Use(middleware.Accounting(), gin.Recovery())

	// 5. Rutas públicas.
	r.POST("/api/vendedor/registro", handlers.RegistrarVendedor)

	// 6. Rutas del vendedor.
	//    Más adelante se les agregará autenticación y autorización.
	rutasVendedor := r.Group("/api/vendedor")
	{
		rutasVendedor.POST("/vehiculos", handlers.PublicarVehiculo)
		rutasVendedor.PUT("/vehiculos/:id/venta", handlers.ReportarVenta)
	}

	// 7. Iniciar el servidor. Si falla (por ejemplo, el puerto está ocupado),
	//    el programa termina mostrando el motivo.
	log.Println("AutoMarket escuchando en http://localhost:8080")
	if err := r.Run(":8080"); err != nil {
		log.Fatalf("no se pudo iniciar el servidor: %v", err)
	}
}
