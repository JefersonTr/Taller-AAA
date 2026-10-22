// Punto de entrada de AutoMarket: configura el servidor y registra las rutas.
package main

import (
	"log"
	"os"
	"path/filepath"

	"automarket/auth"
	"automarket/handlers"
	"automarket/imagenes"
	"automarket/middleware"
	"automarket/models"
	"automarket/repository"

	"github.com/gin-gonic/gin"
)

// Archivos de datos persistentes. La carpeta "datos" está en .gitignore:
// nunca se sube al repositorio.
var (
	rutaAuditoria = filepath.Join("datos", "auditoria.jsonl")
	rutaDatos     = filepath.Join("datos", "datos.json")
	rutaImagenes  = filepath.Join("datos", "imagenes")
)

func main() {
	// 1. Crear la base de datos en memoria.
	repository.InicializarBD()

	// 2. Cargar los datos guardados: la auditoría (verificando su integridad),
	//    los usuarios y vehículos, y la carpeta de fotos.
	//    Si algo falla, el servidor NO arranca.
	if err := repository.DB.AbrirArchivoAuditoria(rutaAuditoria); err != nil {
		log.Fatalf("no se pudo cargar el registro de auditoría: %v", err)
	}
	if err := repository.DB.AbrirArchivoDatos(rutaDatos); err != nil {
		log.Fatalf("no se pudieron cargar los datos: %v", err)
	}
	if err := imagenes.Configurar(rutaImagenes); err != nil {
		log.Fatalf("no se pudo preparar la carpeta de imágenes: %v", err)
	}

	// 3. Crear el administrador inicial, si todavía no existe.
	crearAdministradorInicial()

	// 4. Crear el servicio de autenticación con el repositorio real.
	servicioAutenticacion := auth.NuevoServicio(repository.DB)

	// 5. Crear el servidor SIN middlewares predeterminados.
	//    gin.Default() agrega su propio logger, que duplicaría el Accounting.
	r := gin.New()

	// No confiar en ningún proxy. Así c.ClientIP() usa la IP real de la
	// conexión y no un header X-Forwarded-For que el cliente puede falsificar.
	if err := r.SetTrustedProxies(nil); err != nil {
		log.Fatalf("no se pudo configurar los proxies de confianza: %v", err)
	}

	// Middlewares globales. El orden importa:
	//   - Accounting primero, para registrar TODAS las operaciones,
	//     incluidas las que rechacen los demás.
	//   - Recovery: si un handler falla de forma inesperada, responde 500
	//     y Accounting alcanza a registrarlo.
	//   - ProteccionCSRF: rechaza formularios enviados desde otros sitios.
	r.Use(middleware.Accounting(), gin.Recovery(), middleware.ProteccionCSRF())

	// 6. Frontend: plantillas HTML, hoja de estilos y fotos de los vehículos.
	//    r.Static no muestra la lista de archivos de la carpeta y no permite
	//    salir de ella con rutas como "../".
	handlers.ConfigurarPlantillas(r, "plantillas/*.html")
	r.Static("/estaticos", "./estaticos")
	r.Static("/imagenes", rutaImagenes)

	// 7. API pública: registro de vendedores y catálogo (HU-VEN-01, HU-VIS-01).
	r.POST("/api/vendedor/registro", handlers.RegistrarVendedor)
	r.GET("/api/vehiculos", handlers.ListarCatalogo)
	r.GET("/api/vehiculos/:id", handlers.ConsultarVehiculo)

	// 8. Páginas públicas: catálogo, detalle y registro de vendedores.
	r.GET("/", handlers.PaginaCatalogo)
	r.GET("/vehiculos/:id", handlers.PaginaVehiculo)
	r.GET("/registro", handlers.PaginaRegistro)
	r.POST("/registro", handlers.EnviarRegistro)

	// 9. Páginas del vendedor (HU-VEN-01): mismos middlewares que su API.
	//    Al entrar sin credenciales, el navegador muestra su ventana de login.
	paginasVendedor := r.Group("/vendedor")
	paginasVendedor.Use(
		middleware.RequiereAutenticacion(servicioAutenticacion),
		middleware.RequiereRol(models.RolVendedor),
	)
	{
		paginasVendedor.GET("", handlers.PaginaVendedor)
		paginasVendedor.POST("/vehiculos", handlers.EnviarPublicacion)
		paginasVendedor.POST("/vehiculos/:id/venta", handlers.EnviarVenta)
	}

	// Páginas del administrador (HU-ADM-01): mismos middlewares que su API.
	paginasAdmin := r.Group("/admin")
	paginasAdmin.Use(
		middleware.RequiereAutenticacion(servicioAutenticacion),
		middleware.RequiereRol(models.RolAdministrador),
	)
	{
		paginasAdmin.GET("", handlers.PaginaModeracion)
		paginasAdmin.POST("/vehiculos/:id/autorizacion", handlers.EnviarAutorizacion)
		paginasAdmin.POST("/vehiculos/:id/eliminacion", handlers.EnviarEliminacionVehiculo)
		paginasAdmin.POST("/usuarios/:id/eliminacion", handlers.EnviarEliminacionUsuario)
	}

	// 10. API del vendedor: exige estar autenticado y tener rol vendedor.
	rutasVendedor := r.Group("/api/vendedor")
	rutasVendedor.Use(
		middleware.RequiereAutenticacion(servicioAutenticacion), // ¿Quién eres? (401)
		middleware.RequiereRol(models.RolVendedor),              // ¿Puedes hacer esto? (403)
	)
	{
		rutasVendedor.POST("/vehiculos", handlers.PublicarVehiculo)
		rutasVendedor.PUT("/vehiculos/:id/venta", handlers.ReportarVenta)
	}

	// 11. API del administrador (HU-ADM-01): exige rol administrador.
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

	// Direcciones inexistentes: página 404 para el navegador, JSON para la API.
	r.NoRoute(handlers.RutaNoEncontrada)

	// 12. Iniciar el servidor. Si falla (por ejemplo, el puerto está ocupado),
	//     el programa termina mostrando el motivo.
	log.Println("AutoMarket escuchando en http://localhost:8080")
	if err := r.Run(":8080"); err != nil {
		log.Fatalf("no se pudo iniciar el servidor: %v", err)
	}
}

// crearAdministradorInicial registra al administrador del sistema con las
// credenciales de las variables de entorno AUTOMARKET_ADMIN_USUARIO y
// AUTOMARKET_ADMIN_CONTRASENA, si todavía no existe en los datos guardados.
func crearAdministradorInicial() {
	nombreUsuario := os.Getenv("AUTOMARKET_ADMIN_USUARIO")
	contrasena := os.Getenv("AUTOMARKET_ADMIN_CONTRASENA")
	if nombreUsuario == "" || contrasena == "" {
		log.Println("aviso: variables del administrador no definidas; se usan solo los usuarios guardados")
		return
	}

	// Con la persistencia, el administrador ya puede existir desde una
	// ejecución anterior. Si existe, se verifica que realmente sea
	// administrador: si alguien hubiera registrado ese nombre como vendedor
	// (por ejemplo, mientras el servidor corría sin estas variables), el
	// servidor se detiene en lugar de arrancar sin administrador.
	if existente, err := repository.DB.BuscarUsuarioPorNombreUsuario(nombreUsuario); err == nil {
		if existente.Rol != models.RolAdministrador {
			log.Fatalf("el usuario %q ya existe y no es administrador", nombreUsuario)
		}
		log.Printf("administrador %q cargado desde los datos guardados", nombreUsuario)
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
