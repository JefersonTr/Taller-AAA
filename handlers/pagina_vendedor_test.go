package handlers

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"automarket/middleware"
	"automarket/models"
	"automarket/repository"

	"github.com/gin-gonic/gin"
)

// prepararPaginasVendedor crea a Ana (ID 1) con un vehículo publicado (ID 1)
// y uno pendiente (ID 2), y a Luis (ID 2) con un Kia publicado (ID 3).
func prepararPaginasVendedor(t *testing.T) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	repository.InicializarBD()

	ana, _ := repository.DB.GuardarUsuario(models.Usuario{
		Nombre: "Ana Gómez", NombreUsuario: "ana", Rol: models.RolVendedor,
	})
	luis, _ := repository.DB.GuardarUsuario(models.Usuario{
		Nombre: "Luis Pérez", NombreUsuario: "luis", Rol: models.RolVendedor,
	})
	repository.DB.GuardarVehiculo(models.Vehiculo{
		Marca: "Renault", Modelo: "Koleos 2018", Precio: 78000000,
		Estado: models.EstadoPublicada, VendedorID: ana.ID,
	})
	repository.DB.GuardarVehiculo(models.Vehiculo{
		Marca: "Mazda", Modelo: "3 2020", Precio: 65000000,
		Estado: models.EstadoPendiente, VendedorID: ana.ID,
	})
	repository.DB.GuardarVehiculo(models.Vehiculo{
		Marca: "Kia", Modelo: "Picanto 2019", Precio: 40000000,
		Estado: models.EstadoPublicada, VendedorID: luis.ID,
	})

	// Misma configuración que en main.go.
	r := gin.New()
	ConfigurarPlantillas(r, "../plantillas/*.html")
	paginas := r.Group("/vendedor")
	paginas.Use(
		middleware.RequiereAutenticacion(autenticadorPrueba{"ana": ana, "luis": luis}),
		middleware.RequiereRol(models.RolVendedor),
	)
	paginas.GET("", PaginaVendedor)
	paginas.POST("/vehiculos", EnviarPublicacion)
	paginas.POST("/vehiculos/:id/venta", EnviarVenta)
	return r
}

// peticionComo simula una petición del navegador con las credenciales del
// usuario indicado (vacío = sin credenciales) y, si hay, un formulario.
func peticionComo(r *gin.Engine, metodo, ruta, usuario string, valores url.Values) *httptest.ResponseRecorder {
	peticion := httptest.NewRequest(metodo, ruta, strings.NewReader(valores.Encode()))
	peticion.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if usuario != "" {
		peticion.SetBasicAuth(usuario, "clave")
	}
	respuesta := httptest.NewRecorder()
	r.ServeHTTP(respuesta, peticion)
	return respuesta
}

func TestPaginaVendedor(t *testing.T) {
	r := prepararPaginasVendedor(t)

	// Caso 1: sin credenciales, 401 con la cabecera que abre la ventana de login.
	respuesta := peticionComo(r, http.MethodGet, "/vendedor", "", nil)
	if respuesta.Code != http.StatusUnauthorized || respuesta.Header().Get("WWW-Authenticate") == "" {
		t.Fatal("sin credenciales debe responder 401 con WWW-Authenticate")
	}

	// Caso 2: Ana ve SUS vehículos, en cualquier estado, y no los de Luis.
	cuerpo := peticionComo(r, http.MethodGet, "/vendedor", "ana", nil).Body.String()
	if !strings.Contains(cuerpo, "Koleos 2018") || !strings.Contains(cuerpo, "3 2020") {
		t.Fatal("Ana debe ver sus vehículos publicados y pendientes")
	}
	if strings.Contains(cuerpo, "Picanto") {
		t.Fatal("Ana no debe ver los vehículos de Luis")
	}
}

func TestEnviarPublicacion(t *testing.T) {
	r := prepararPaginasVendedor(t)

	// Caso 1: publicación válida. Redirige con aviso y queda pendiente,
	// asociada a Ana (ID 1), aunque el formulario intente otra cosa.
	valido := url.Values{
		"marca": {"Chevrolet"}, "modelo": {"Onix 2021"}, "precio": {"55000000"},
		"estado": {"publicada"}, "vendedor_id": {"2"},
	}
	respuesta := peticionComo(r, http.MethodPost, "/vendedor/vehiculos", "ana", valido)
	if respuesta.Code != http.StatusSeeOther || respuesta.Header().Get("Location") != "/vendedor?aviso=publicado" {
		t.Fatalf("debería redirigir con aviso; obtuvo %d %q", respuesta.Code, respuesta.Header().Get("Location"))
	}
	nuevo, err := repository.DB.BuscarVehiculoPorID(4)
	if err != nil || nuevo.Estado != models.EstadoPendiente || nuevo.VendedorID != 1 {
		t.Fatal("la publicación debe quedar pendiente y asociada a Ana")
	}

	// Caso 2: precio que no es número. Se muestra el error en la página.
	invalido := url.Values{"marca": {"Chevrolet"}, "modelo": {"Onix"}, "precio": {"abc"}}
	respuesta = peticionComo(r, http.MethodPost, "/vendedor/vehiculos", "ana", invalido)
	if respuesta.Code != http.StatusBadRequest || !strings.Contains(respuesta.Body.String(), "mayor que cero") {
		t.Fatalf("debería mostrar el error del precio; obtuvo %d", respuesta.Code)
	}
}

func TestEnviarVenta(t *testing.T) {
	r := prepararPaginasVendedor(t)

	// El ORDEN importa: la venta exitosa cambia el estado del vehículo 1.
	casos := []struct {
		nombre         string
		usuario        string
		ruta           string
		codigoEsperado int
	}{
		{"vehículo de otro vendedor", "luis", "/vendedor/vehiculos/1/venta", http.StatusForbidden},
		{"vehículo pendiente", "ana", "/vendedor/vehiculos/2/venta", http.StatusConflict},
		{"vehículo inexistente", "ana", "/vendedor/vehiculos/99/venta", http.StatusNotFound},
		{"venta exitosa", "ana", "/vendedor/vehiculos/1/venta", http.StatusSeeOther},
		{"vehículo ya vendido", "ana", "/vendedor/vehiculos/1/venta", http.StatusConflict},
	}

	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			respuesta := peticionComo(r, http.MethodPost, caso.ruta, caso.usuario, nil)
			if respuesta.Code != caso.codigoEsperado {
				t.Fatalf("código esperado %d, obtenido %d", caso.codigoEsperado, respuesta.Code)
			}
		})
	}
}
