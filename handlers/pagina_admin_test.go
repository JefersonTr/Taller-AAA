package handlers

import (
	"net/http"
	"strings"
	"testing"

	"automarket/middleware"
	"automarket/models"
	"automarket/repository"

	"github.com/gin-gonic/gin"
)

// prepararPaginasAdmin crea al administrador (usuario 1), a Ana (usuario 2),
// un Mazda pendiente (vehículo 1) y un Renault publicado (vehículo 2).
func prepararPaginasAdmin(t *testing.T) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	repository.InicializarBD()

	admin, _ := repository.DB.GuardarUsuario(models.Usuario{
		Nombre: "Administrador", NombreUsuario: "admin", Rol: models.RolAdministrador,
	})
	ana, _ := repository.DB.GuardarUsuario(models.Usuario{
		Nombre: "Ana Gómez", NombreUsuario: "ana", Telefono: "3001234567", Rol: models.RolVendedor,
	})
	repository.DB.GuardarVehiculo(models.Vehiculo{
		Marca: "Mazda", Modelo: "3 2020", Precio: 65000000,
		Estado: models.EstadoPendiente, VendedorID: ana.ID,
	})
	repository.DB.GuardarVehiculo(models.Vehiculo{
		Marca: "Renault", Modelo: "Koleos 2018", Precio: 78000000,
		Estado: models.EstadoPublicada, VendedorID: ana.ID,
	})

	// Misma configuración que en main.go.
	r := gin.New()
	ConfigurarPlantillas(r, "../plantillas/*.html")
	paginas := r.Group("/admin")
	paginas.Use(
		middleware.RequiereAutenticacion(autenticadorPrueba{"admin": admin, "ana": ana}),
		middleware.RequiereRol(models.RolAdministrador),
	)
	paginas.GET("", PaginaModeracion)
	paginas.POST("/vehiculos/:id/autorizacion", EnviarAutorizacion)
	paginas.POST("/vehiculos/:id/eliminacion", EnviarEliminacionVehiculo)
	paginas.POST("/usuarios/:id/eliminacion", EnviarEliminacionUsuario)
	return r
}

func TestPaginaModeracion(t *testing.T) {
	r := prepararPaginasAdmin(t)

	// Caso 1: una vendedora no puede entrar.
	if codigo := peticionComo(r, http.MethodGet, "/admin", "ana", nil).Code; codigo != http.StatusForbidden {
		t.Fatalf("una vendedora debería recibir 403; obtuvo %d", codigo)
	}

	// Caso 2: el administrador ve las dos listas y a la vendedora.
	cuerpo := peticionComo(r, http.MethodGet, "/admin", "admin", nil).Body.String()
	for _, esperado := range []string{"Mazda", "Renault", "Ana Gómez", "3001234567"} {
		if !strings.Contains(cuerpo, esperado) {
			t.Fatalf("el panel debería mostrar %q", esperado)
		}
	}
}

func TestAccionesAdmin(t *testing.T) {
	r := prepararPaginasAdmin(t)

	// El ORDEN importa: cada acción cambia los datos para la siguiente.
	casos := []struct {
		nombre          string
		ruta            string
		codigoEsperado  int
		destinoEsperado string // A dónde redirige si el código es 303
	}{
		{"autorizar pendiente", "/admin/vehiculos/1/autorizacion", http.StatusSeeOther, "/admin?aviso=autorizado"},
		{"autorizar otra vez", "/admin/vehiculos/1/autorizacion", http.StatusConflict, ""},
		{"eliminar publicación", "/admin/vehiculos/2/eliminacion", http.StatusSeeOther, "/admin?aviso=publicacion-eliminada"},
		{"eliminar al administrador", "/admin/usuarios/1/eliminacion", http.StatusForbidden, ""},
		{"eliminar a la vendedora", "/admin/usuarios/2/eliminacion", http.StatusSeeOther, "/admin?aviso=usuario-eliminado"},
		{"eliminar usuario inexistente", "/admin/usuarios/99/eliminacion", http.StatusNotFound, ""},
	}

	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			respuesta := peticionComo(r, http.MethodPost, caso.ruta, "admin", nil)
			if respuesta.Code != caso.codigoEsperado {
				t.Fatalf("código esperado %d, obtenido %d", caso.codigoEsperado, respuesta.Code)
			}
			if caso.destinoEsperado != "" && respuesta.Header().Get("Location") != caso.destinoEsperado {
				t.Fatalf("debería redirigir a %q; redirige a %q",
					caso.destinoEsperado, respuesta.Header().Get("Location"))
			}
		})
	}

	// Al eliminar a Ana, su publicación autorizada (vehículo 1) también desapareció.
	if _, err := repository.DB.BuscarVehiculoPorID(1); err == nil {
		t.Fatal("las publicaciones de la vendedora eliminada también deberían eliminarse")
	}
}
