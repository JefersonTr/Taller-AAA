package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"automarket/auth"
	"automarket/middleware"
	"automarket/models"
	"automarket/repository"

	"github.com/gin-gonic/gin"
)

// autenticadorPrueba acepta a los usuarios del mapa con la contraseña "clave".
// Cumple la interfaz middleware.Autenticador.
type autenticadorPrueba map[string]models.Usuario

func (a autenticadorPrueba) Autenticar(nombreUsuario, contrasena string) (models.Usuario, error) {
	usuario, existe := a[nombreUsuario]
	if !existe || contrasena != "clave" {
		return models.Usuario{}, auth.ErrCredencialesInvalidas
	}
	return usuario, nil
}

func TestReportarVenta(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repository.InicializarBD()

	// Dos vendedores: Ana es dueña de los vehículos, Luis no.
	ana := models.Usuario{ID: 1, NombreUsuario: "ana", Rol: models.RolVendedor}
	luis := models.Usuario{ID: 2, NombreUsuario: "luis", Rol: models.RolVendedor}

	// Vehículo 1: publicado. Vehículo 2: pendiente. Ambos de Ana.
	repository.DB.GuardarVehiculo(models.Vehiculo{
		Marca: "Renault", Modelo: "Koleos 2018", Precio: 78000000,
		Estado: models.EstadoPublicada, VendedorID: ana.ID,
	})
	repository.DB.GuardarVehiculo(models.Vehiculo{
		Marca: "Mazda", Modelo: "3 2020", Precio: 65000000,
		Estado: models.EstadoPendiente, VendedorID: ana.ID,
	})

	// Misma cadena de middlewares que en main.go.
	r := gin.New()
	r.PUT("/vehiculos/:id/venta",
		middleware.RequiereAutenticacion(autenticadorPrueba{"ana": ana, "luis": luis}),
		middleware.RequiereRol(models.RolVendedor),
		ReportarVenta,
	)

	// El ORDEN de los casos importa: "venta exitosa" cambia el estado del
	// vehículo 1, y el último caso comprueba que no se pueda vender dos veces.
	casos := []struct {
		nombre         string
		usuario        string
		ruta           string
		codigoEsperado int
	}{
		{"vendedor que no es dueño", "luis", "/vehiculos/1/venta", http.StatusForbidden},
		{"vehículo pendiente", "ana", "/vehiculos/2/venta", http.StatusConflict},
		{"vehículo inexistente", "ana", "/vehiculos/99/venta", http.StatusNotFound},
		{"identificador inválido", "ana", "/vehiculos/abc/venta", http.StatusBadRequest},
		{"venta exitosa", "ana", "/vehiculos/1/venta", http.StatusOK},
		{"vehículo ya vendido", "ana", "/vehiculos/1/venta", http.StatusConflict},
	}

	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			peticion := httptest.NewRequest(http.MethodPut, caso.ruta, nil)
			peticion.SetBasicAuth(caso.usuario, "clave")
			respuesta := httptest.NewRecorder()
			r.ServeHTTP(respuesta, peticion)

			if respuesta.Code != caso.codigoEsperado {
				t.Fatalf("código esperado %d, obtenido %d: %s",
					caso.codigoEsperado, respuesta.Code, respuesta.Body.String())
			}
		})
	}

	// La venta debe quedar guardada en el repositorio.
	vehiculo, _ := repository.DB.BuscarVehiculoPorID(1)
	if vehiculo.Estado != models.EstadoVendido {
		t.Fatal("el vehículo vendido debe quedar en estado vendido")
	}
}
