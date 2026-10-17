package handlers

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"automarket/models"
	"automarket/repository"

	"github.com/gin-gonic/gin"
)

// prepararRegistro crea una base de datos vacía y un router con las
// páginas de registro y del catálogo.
func prepararRegistro(t *testing.T) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	repository.InicializarBD()

	r := gin.New()
	ConfigurarPlantillas(r, "../plantillas/*.html")
	r.GET("/", PaginaCatalogo)
	r.GET("/registro", PaginaRegistro)
	r.POST("/registro", EnviarRegistro)
	return r
}

// enviarFormulario simula el envío de un formulario HTML.
func enviarFormulario(r *gin.Engine, ruta string, valores url.Values) *httptest.ResponseRecorder {
	peticion := httptest.NewRequest(http.MethodPost, ruta, strings.NewReader(valores.Encode()))
	// Este es el formato en que los navegadores envían los formularios.
	peticion.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	respuesta := httptest.NewRecorder()
	r.ServeHTTP(respuesta, peticion)
	return respuesta
}

func TestPaginaRegistro(t *testing.T) {
	r := prepararRegistro(t)

	// Caso 1: el formulario se muestra.
	if respuesta := hacerPeticion(r, "/registro"); respuesta.Code != http.StatusOK ||
		!strings.Contains(respuesta.Body.String(), "<form") {
		t.Fatal("la página de registro debe mostrar el formulario")
	}

	// Caso 2: registro válido. El campo "rol" extra se ignora.
	valido := url.Values{
		"nombre": {"Ana Gómez"}, "nombre_usuario": {"ana"},
		"telefono": {"3001234567"}, "contrasena": {"secreta123"},
		"rol": {"administrador"},
	}
	respuesta := enviarFormulario(r, "/registro", valido)
	if respuesta.Code != http.StatusSeeOther || respuesta.Header().Get("Location") != "/?aviso=registrado" {
		t.Fatalf("debería redirigir con 303 al catálogo con aviso; obtuvo %d %q",
			respuesta.Code, respuesta.Header().Get("Location"))
	}
	ana, err := repository.DB.BuscarUsuarioPorNombreUsuario("ana")
	if err != nil || ana.Rol != models.RolVendedor {
		t.Fatal("debería registrar a Ana como vendedora, sin importar el campo rol")
	}

	// Caso 3: usuario repetido. Se muestra el error, se conservan los datos
	// escritos y NO se devuelve la contraseña.
	respuesta = enviarFormulario(r, "/registro", valido)
	cuerpo := respuesta.Body.String()
	if respuesta.Code != http.StatusConflict || !strings.Contains(cuerpo, "ya está registrado") {
		t.Fatalf("debería mostrar el error de usuario repetido; obtuvo %d", respuesta.Code)
	}
	if !strings.Contains(cuerpo, "Ana Gómez") {
		t.Fatal("el formulario debería conservar los datos escritos")
	}
	if strings.Contains(cuerpo, "secreta123") {
		t.Fatal("la contraseña nunca debe devolverse a la página")
	}

	// Caso 4: falta la contraseña.
	sinContrasena := url.Values{
		"nombre": {"Luis"}, "nombre_usuario": {"luis"}, "telefono": {"3000000000"},
	}
	if respuesta := enviarFormulario(r, "/registro", sinContrasena); respuesta.Code != http.StatusBadRequest {
		t.Fatalf("debería rechazar el registro sin contraseña; obtuvo %d", respuesta.Code)
	}
}

func TestAvisos(t *testing.T) {
	r := prepararRegistro(t)

	// Un código conocido muestra su texto.
	if !strings.Contains(hacerPeticion(r, "/?aviso=registrado").Body.String(), "Tu cuenta fue creada") {
		t.Fatal("el aviso 'registrado' debería mostrarse")
	}

	// Un texto arbitrario en la URL NO se muestra (protección contra phishing).
	if strings.Contains(hacerPeticion(r, "/?aviso=Tu+cuenta+fue+suspendida").Body.String(), "suspendida") {
		t.Fatal("un aviso desconocido no debe mostrarse")
	}
}
