package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"automarket/models"
	"automarket/repository"

	"github.com/gin-gonic/gin"
)

// prepararCatalogo crea una base de datos limpia con un vendedor,
// un vehículo publicado (ID 1) y uno pendiente (ID 2), y devuelve
// un router con las rutas del visitante.
func prepararCatalogo(t *testing.T) *gin.Engine {
	gin.SetMode(gin.TestMode)
	repository.InicializarBD()

	vendedor, err := repository.DB.GuardarUsuario(models.Usuario{
		Nombre:         "Ana Gómez",
		NombreUsuario:  "ana",
		Telefono:       "3001234567",
		Rol:            models.RolVendedor,
		HashContrasena: "sal$hash",
	})
	if err != nil {
		t.Fatal(err)
	}

	repository.DB.GuardarVehiculo(models.Vehiculo{
		Marca: "Renault", Modelo: "Koleos 2018", Precio: 78000000,
		Estado: models.EstadoPublicada, VendedorID: vendedor.ID,
	})
	repository.DB.GuardarVehiculo(models.Vehiculo{
		Marca: "Mazda", Modelo: "3 2020", Precio: 65000000,
		Estado: models.EstadoPendiente, VendedorID: vendedor.ID,
	})

	r := gin.New()
	r.GET("/api/vehiculos", ListarCatalogo)
	r.GET("/api/vehiculos/:id", ConsultarVehiculo)
	return r
}

// hacerPeticion ejecuta un GET sobre el router y devuelve la respuesta.
func hacerPeticion(r *gin.Engine, ruta string) *httptest.ResponseRecorder {
	respuesta := httptest.NewRecorder()
	r.ServeHTTP(respuesta, httptest.NewRequest(http.MethodGet, ruta, nil))
	return respuesta
}

func TestListarCatalogo(t *testing.T) {
	r := prepararCatalogo(t)

	respuesta := hacerPeticion(r, "/api/vehiculos")
	if respuesta.Code != http.StatusOK {
		t.Fatalf("código esperado 200, obtenido %d", respuesta.Code)
	}

	// Convertir el JSON de la respuesta a la estructura de Go.
	var catalogo []vehiculoPublico
	if err := json.Unmarshal(respuesta.Body.Bytes(), &catalogo); err != nil {
		t.Fatal("la respuesta no es un JSON válido:", err)
	}

	if len(catalogo) != 1 || catalogo[0].ID != 1 {
		t.Fatal("el catálogo solo debe mostrar el vehículo publicado")
	}
	if catalogo[0].Vendedor.Telefono != "3001234567" {
		t.Fatal("cada vehículo debe incluir el contacto del vendedor")
	}
}

func TestConsultarVehiculo(t *testing.T) {
	r := prepararCatalogo(t)

	casos := []struct {
		nombre         string
		ruta           string
		codigoEsperado int
	}{
		{"vehículo publicado", "/api/vehiculos/1", http.StatusOK},
		{"vehículo pendiente", "/api/vehiculos/2", http.StatusNotFound},
		{"vehículo inexistente", "/api/vehiculos/99", http.StatusNotFound},
		{"identificador inválido", "/api/vehiculos/abc", http.StatusBadRequest},
	}

	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			respuesta := hacerPeticion(r, caso.ruta)
			if respuesta.Code != caso.codigoEsperado {
				t.Fatalf("código esperado %d, obtenido %d", caso.codigoEsperado, respuesta.Code)
			}
		})
	}

	// La respuesta pública nunca debe contener datos sensibles del vendedor.
	cuerpo := hacerPeticion(r, "/api/vehiculos/1").Body.String()
	for _, prohibido := range []string{"sal$hash", "nombre_usuario", "\"rol\""} {
		if strings.Contains(cuerpo, prohibido) {
			t.Fatalf("la respuesta pública expone un dato sensible: %s", prohibido)
		}
	}
}
