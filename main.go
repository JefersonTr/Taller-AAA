// Punto de entrada de AutoMarket: configura el servidor y registra las rutas.
package main

import (
	"log"
	"os"
	"path/filepath"

	"automarket/auth"
	"automarket/handlers"
	"automarket/middleware"
	"automarket/models"
	"automarket/repository"

	"github.com/gin-gonic/gin"
)

// rutaAuditoria es el archivo del registro de auditoría persistente.
// La carpeta "datos" está en .gitignore: nunca se sube al repositorio.
var rutaAuditoria = filepath.Join("datos", "auditoria.jsonl")

func main() {
	// 1. Crear la base de datos en memoria y el administrador inicial.
	repository.InicializarBD()
	// Cargar el registro de auditoría y verificar su integridad.
	// Si fue alterado, el servidor NO arranca ("fallar cerrado").
	if err := repository.DB.AbrirArchivoAuditoria(rutaAuditoria); err != nil {
		log.Fatalf("no se pudo cargar el registro de auditoría: %v", err)
	}
	crearAdministradorInicial()

	// 2. Crear el servicio de autenticación con el repositorio real.
	//    repository.DB cumple la interfaz RepositorioUsuarios porque tiene
	//    el método BuscarUsuarioPorNombreUsuario.
	servicioAutenticacion := auth.NuevoServicio(repository.DB)

	// 3. Crear el servidor SIN middlewares predeterminados.
	//    gin.Default() agrega su propio logger, que duplicaría el Accounting.
	r := gin.New()

	// 4. No confiar en ningún proxy. Así c.ClientIP() usa la IP real de la
	//    conexión y no un header X-Forwarded-For que el cliente puede falsificar.
	if err := r.SetTrustedProxies(nil); err != nil {
		log.Fatalf("no se pudo configurar los proxies de confianza: %v", err)
	}

	// 5. Middlewares globales. El orden importa:
	//    - Accounting primero, para registrar TODAS las operaciones.
	//    - Recovery después: si un handler falla de forma inesperada,
	//      Recovery responde 500 y Accounting alcanza a registrarlo.
	r.Use(middleware.Accounting(), gin.Recovery())

	// Frontend: plantillas HTML y archivos estáticos (hoja de estilos).
	handlers.ConfigurarPlantillas(r, "plantillas/*.html")
	r.Static("/estaticos", "./estaticos")

	// 6. Rutas públicas.
	//    El registro se declara directamente en "r" y no en el grupo de abajo,
	//    por eso NO le aplican la autenticación ni la autorización del grupo.
	r.POST("/api/vendedor/registro", handlers.RegistrarVendedor)

	// Catálogo del visitante (HU-VIS-01): no exige autenticación.
	r.GET("/api/vehiculos", handlers.ListarCatalogo)
	r.GET("/api/vehiculos/:id", handlers.ConsultarVehiculo)

	// Páginas públicas del visitante (HU-VIS-01).
	r.GET("/", handlers.PaginaCatalogo)
	r.GET("/vehiculos/:id", handlers.PaginaVehiculo)

	// 7. Rutas del vendedor: exigen estar autenticado y tener rol vendedor.
	rutasVendedor := r.Group("/api/vendedor")
	rutasVendedor.Use(
		middleware.RequiereAutenticacion(servicioAutenticacion), // ¿Quién eres? (401)
		middleware.RequiereRol(models.RolVendedor),              // ¿Puedes hacer esto? (403)
	)
	{
		rutasVendedor.POST("/vehiculos", handlers.PublicarVehiculo)
		rutasVendedor.PUT("/vehiculos/:id/venta", handlers.ReportarVenta)
	}

	// Rutas del administrador (HU-ADM-01): exigen estar autenticado
	// y tener rol administrador.
	rutasAdmin := r.Group("/api/admin")
	rutasAdmin.Use(
		middleware.RequiereAutenticacion(servicioAutenticacion), // ¿Quién eres? (401)
		middleware.RequiereRol(models.RolAdministrador),         // ¿Puedes hacer esto? (403)
	)
	{
		rutasAdmin.GET("/vehiculos/pendientes", handlers.ListarPendientes)
		rutasAdmin.PUT("/vehiculos/:id/autorizacion", handlers.AutorizarPublicacion)
		rutasAdmin.DELETE("/vehiculos/:id", handlers.EliminarPublicacion)
		rutasAdmin.DELETE("/usuarios/:id", handlers.EliminarUsuario)
	}

	// 8. Iniciar el servidor. Si falla (por ejemplo, el puerto está ocupado),
	//    el programa termina mostrando el motivo.
	log.Println("AutoMarket escuchando en http://localhost:8080")
	if err := r.Run(":8080"); err != nil {
		log.Fatalf("no se pudo iniciar el servidor: %v", err)
	}
}

// crearAdministradorInicial registra al administrador del sistema con las
// credenciales de las variables de entorno AUTOMARKET_ADMIN_USUARIO y
// AUTOMARKET_ADMIN_CONTRASENA.
//
// Si no están definidas, el servidor arranca sin administrador y lo avisa.
func crearAdministradorInicial() {
	nombreUsuario := os.Getenv("AUTOMARKET_ADMIN_USUARIO")
	contrasena := os.Getenv("AUTOMARKET_ADMIN_CONTRASENA")
	if nombreUsuario == "" || contrasena == "" {
		log.Println("aviso: no se creó el administrador; defina AUTOMARKET_ADMIN_USUARIO y AUTOMARKET_ADMIN_CONTRASENA")
		return
	}

	hash, err := auth.GenerarHash(contrasena)
	if err != nil {
		log.Fatalf("no se pudo generar el hash del administrador: %v", err)
	}

	_, err = repository.DB.GuardarUsuario(models.Usuario{
		Nombre:         "Administrador",
		NombreUsuario:  nombreUsuario,
		Rol:            models.RolAdministrador,
		HashContrasena: hash,
	})
	if err != nil {
		log.Fatalf("no se pudo crear el administrador: %v", err)
	}
	log.Printf("administrador %q creado", nombreUsuario)
}
