package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"automarket/middleware"
	"automarket/models"
	"automarket/repository"

	"github.com/gin-gonic/gin"
)

func TestAdministrador(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repository.InicializarBD()

	// Datos: un administrador (ID 1) y una vendedora (ID 2) con dos
	// vehículos: uno pendiente (ID 1) y uno publicado (ID 2).
	admin, _ := repository.DB.GuardarUsuario(models.Usuario{
		NombreUsuario: "admin", Rol: models.RolAdministrador,
	})
	ana, _ := repository.DB.GuardarUsuario(models.Usuario{
		NombreUsuario: "ana", Rol: models.RolVendedor,
	})
	repository.DB.GuardarVehiculo(models.Vehiculo{
		Marca: "Mazda", Estado: models.EstadoPendiente, VendedorID: ana.ID,
	})
	repository.DB.GuardarVehiculo(models.Vehiculo{
		Marca: "Renault", Estado: models.EstadoPublicada, VendedorID: ana.ID,
	})

	// Misma cadena de middlewares que en main.go.
	r := gin.New()
	rutas := r.Group("/admin")
	rutas.Use(
		middleware.RequiereAutenticacion(autenticadorPrueba{"admin": admin, "ana": ana}),
		middleware.RequiereRol(models.RolAdministrador),
	)
	rutas.GET("/vehiculos/pendientes", ListarPendientes)
	rutas.PUT("/vehiculos/:id/autorizacion", AutorizarPublicacion)
	rutas.DELETE("/vehiculos/:id", EliminarPublicacion)
	rutas.DELETE("/usuarios/:id", EliminarUsuario)

	// El ORDEN importa: varios casos dependen de lo que hicieron los anteriores.
	// Un usuario vacío ("") significa que la petición va sin credenciales.
	casos := []struct {
		nombre         string
		metodo         string
		ruta           string
		usuario        string
		codigoEsperado int
	}{
		{"sin credenciales", http.MethodGet, "/admin/vehiculos/pendientes", "", http.StatusUnauthorized},
		{"vendedor en ruta de administrador", http.MethodGet, "/admin/vehiculos/pendientes", "ana", http.StatusForbidden},
		{"listar pendientes", http.MethodGet, "/admin/vehiculos/pendientes", "admin", http.StatusOK},
		{"autorizar publicación pendiente", http.MethodPut, "/admin/vehiculos/1/autorizacion", "admin", http.StatusOK},
		{"autorizar publicación ya publicada", http.MethodPut, "/admin/vehiculos/1/autorizacion", "admin", http.StatusConflict},
		{"autorizar publicación inexistente", http.MethodPut, "/admin/vehiculos/99/autorizacion", "admin", http.StatusNotFound},
		{"identificador inválido", http.MethodPut, "/admin/vehiculos/abc/autorizacion", "admin", http.StatusBadRequest},
		{"eliminar publicación", http.MethodDelete, "/admin/vehiculos/2", "admin", http.StatusOK},
		{"eliminar publicación ya eliminada", http.MethodDelete, "/admin/vehiculos/2", "admin", http.StatusNotFound},
		{"eliminar a un administrador", http.MethodDelete, "/admin/usuarios/1", "admin", http.StatusForbidden},
		{"eliminar a una vendedora", http.MethodDelete, "/admin/usuarios/2", "admin", http.StatusOK},
		{"eliminar usuario inexistente", http.MethodDelete, "/admin/usuarios/99", "admin", http.StatusNotFound},
	}

	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			peticion := httptest.NewRequest(caso.metodo, caso.ruta, nil)
			if caso.usuario != "" {
				peticion.SetBasicAuth(caso.usuario, "clave")
			}
			respuesta := httptest.NewRecorder()
			r.ServeHTTP(respuesta, peticion)

			if respuesta.Code != caso.codigoEsperado {
				t.Fatalf("código esperado %d, obtenido %d: %s",
					caso.codigoEsperado, respuesta.Code, respuesta.Body.String())
			}
		})
	}

	// Al eliminar a Ana, su publicación autorizada (ID 1) también desapareció.
	if _, err := repository.DB.BuscarVehiculoPorID(1); err == nil {
		t.Fatal("las publicaciones de la vendedora eliminada deberían haberse eliminado")
	}
}
