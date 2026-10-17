package middleware

import (
	"html/template"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

// acceptNavegador es el encabezado Accept típico de un navegador.
const acceptNavegador = "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8"

// routerConPlantillas crea un router con las plantillas reales del sitio.
func routerConPlantillas() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	// Las plantillas usan la función "precio"; para estas pruebas basta
	// con una versión que no hace nada.
	r.SetFuncMap(template.FuncMap{"precio": func(int) string { return "" }})
	r.LoadHTMLGlob("../plantillas/*.html")
	return r
}

// ejecutar envía la petición al router y devuelve la respuesta.
func ejecutar(r *gin.Engine, peticion *http.Request) *httptest.ResponseRecorder {
	respuesta := httptest.NewRecorder()
	r.ServeHTTP(respuesta, peticion)
	return respuesta
}

func TestErrorSegunElCliente(t *testing.T) {
	r := routerConPlantillas()
	r.GET("/protegida", RequiereAutenticacion(autenticadorFalso{}), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	// Caso 1: un navegador recibe la página de error, con la cabecera que
	// abre la ventana de login.
	peticion := httptest.NewRequest(http.MethodGet, "/protegida", nil)
	peticion.Header.Set("Accept", acceptNavegador)
	respuesta := ejecutar(r, peticion)

	if respuesta.Code != http.StatusUnauthorized {
		t.Fatalf("código esperado 401, obtenido %d", respuesta.Code)
	}
	if !strings.HasPrefix(respuesta.Header().Get("Content-Type"), "text/html") ||
		!strings.Contains(respuesta.Body.String(), "autenticación requerida") {
		t.Fatal("un navegador debería recibir la página de error en HTML")
	}
	if respuesta.Header().Get("WWW-Authenticate") == "" {
		t.Fatal("el 401 debe conservar la cabecera WWW-Authenticate")
	}

	// Caso 2: un cliente de la API, sin Accept, sigue recibiendo JSON.
	respuesta = ejecutar(r, httptest.NewRequest(http.MethodGet, "/protegida", nil))
	if !strings.HasPrefix(respuesta.Header().Get("Content-Type"), "application/json") ||
		!strings.Contains(respuesta.Body.String(), `"error"`) {
		t.Fatal("un cliente de la API debería seguir recibiendo JSON")
	}
}

func TestCSRFMuestraPagina(t *testing.T) {
	r := routerConPlantillas()
	r.Use(ProteccionCSRF())
	r.POST("/accion", func(c *gin.Context) { c.Status(http.StatusOK) })

	// Un navegador que envía un formulario desde otro sitio.
	peticion := httptest.NewRequest(http.MethodPost, "/accion", nil)
	peticion.Header.Set("Accept", acceptNavegador)
	peticion.Header.Set("Sec-Fetch-Site", "cross-site")
	respuesta := ejecutar(r, peticion)

	if respuesta.Code != http.StatusForbidden ||
		!strings.Contains(respuesta.Body.String(), "proviene de otro sitio") ||
		!strings.Contains(respuesta.Body.String(), "<html") {
		t.Fatalf("el rechazo CSRF debería mostrarse como página; obtuvo %d", respuesta.Code)
	}
}
