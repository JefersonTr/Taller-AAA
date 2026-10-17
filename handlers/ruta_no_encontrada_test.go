package handlers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestRutaNoEncontrada(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	ConfigurarPlantillas(r, "../plantillas/*.html")
	r.NoRoute(RutaNoEncontrada)

	// Caso 1: un navegador recibe la página 404 del sitio.
	peticion := httptest.NewRequest(http.MethodGet, "/no-existe", nil)
	peticion.Header.Set("Accept", "text/html")
	respuesta := httptest.NewRecorder()
	r.ServeHTTP(respuesta, peticion)

	if respuesta.Code != http.StatusNotFound || !strings.Contains(respuesta.Body.String(), "<html") {
		t.Fatalf("un navegador debería recibir la página 404; obtuvo %d", respuesta.Code)
	}

	// Caso 2: un cliente de la API recibe JSON.
	respuesta = hacerPeticion(r, "/api/no-existe")
	if respuesta.Code != http.StatusNotFound || !strings.Contains(respuesta.Body.String(), `"error"`) {
		t.Fatal("un cliente de la API debería recibir el 404 en JSON")
	}
}
