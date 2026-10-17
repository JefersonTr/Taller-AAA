package handlers

import (
	"net/http"
	"strings"
	"testing"

	"automarket/models"
	"automarket/repository"

	"github.com/gin-gonic/gin"
)

// nombreMalicioso simula a un vendedor que intenta inyectar código
// JavaScript en las páginas a través de su nombre.
const nombreMalicioso = `<script>alert("xss")</script>`

// prepararPaginas crea datos de prueba y un router con las páginas públicas.
// Vehículo 1: publicado. Vehículo 2: pendiente.
func prepararPaginas(t *testing.T) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	repository.InicializarBD()

	vendedor, err := repository.DB.GuardarUsuario(models.Usuario{
		Nombre: nombreMalicioso, NombreUsuario: "ana",
		Telefono: "3001234567", Rol: models.RolVendedor,
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

	// Durante "go test" la carpeta actual es la del paquete (handlers),
	// por eso la ruta sube un nivel con "../".
	r := gin.New()
	ConfigurarPlantillas(r, "../plantillas/*.html")
	r.GET("/", PaginaCatalogo)
	r.GET("/vehiculos/:id", PaginaVehiculo)
	return r
}

func TestPaginaCatalogo(t *testing.T) {
	r := prepararPaginas(t)

	respuesta := hacerPeticion(r, "/")
	cuerpo := respuesta.Body.String()

	if respuesta.Code != http.StatusOK {
		t.Fatalf("código esperado 200, obtenido %d", respuesta.Code)
	}
	if !strings.Contains(cuerpo, "Koleos 2018") || !strings.Contains(cuerpo, "$78.000.000") {
		t.Fatal("el catálogo debe mostrar el vehículo publicado con su precio formateado")
	}
	if strings.Contains(cuerpo, "Mazda") {
		t.Fatal("el catálogo no debe mostrar vehículos pendientes")
	}
}

func TestPaginaVehiculo(t *testing.T) {
	r := prepararPaginas(t)

	casos := []struct {
		nombre         string
		ruta           string
		codigoEsperado int
	}{
		{"vehículo publicado", "/vehiculos/1", http.StatusOK},
		{"vehículo pendiente", "/vehiculos/2", http.StatusNotFound},
		{"vehículo inexistente", "/vehiculos/99", http.StatusNotFound},
		{"identificador inválido", "/vehiculos/abc", http.StatusNotFound},
	}
	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			if codigo := hacerPeticion(r, caso.ruta).Code; codigo != caso.codigoEsperado {
				t.Fatalf("código esperado %d, obtenido %d", caso.codigoEsperado, codigo)
			}
		})
	}

	// El detalle debe mostrar el contacto del vendedor (criterio de HU-VIS-01).
	if !strings.Contains(hacerPeticion(r, "/vehiculos/1").Body.String(), "3001234567") {
		t.Fatal("el detalle debe mostrar el teléfono del vendedor")
	}
}

// TestPaginasEscapanHTML verifica la protección contra XSS: el nombre
// malicioso debe aparecer como texto, nunca como código ejecutable.
func TestPaginasEscapanHTML(t *testing.T) {
	r := prepararPaginas(t)

	for _, ruta := range []string{"/", "/vehiculos/1"} {
		cuerpo := hacerPeticion(r, ruta).Body.String()
		if strings.Contains(cuerpo, "<script>alert") {
			t.Fatalf("la página %s inyectó el script sin escapar", ruta)
		}
		if !strings.Contains(cuerpo, "&lt;script&gt;") {
			t.Fatalf("la página %s debería mostrar el nombre escapado", ruta)
		}
	}
}

func TestFormatearPrecio(t *testing.T) {
	casos := map[int]string{
		0:        "$0",
		999:      "$999",
		1000:     "$1.000",
		65000000: "$65.000.000",
		78000000: "$78.000.000",
	}
	for precio, esperado := range casos {
		if obtenido := FormatearPrecio(precio); obtenido != esperado {
			t.Errorf("FormatearPrecio(%d) = %q, se esperaba %q", precio, obtenido, esperado)
		}
	}
}
