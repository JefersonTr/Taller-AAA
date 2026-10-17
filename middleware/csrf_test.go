package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestProteccionCSRF(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.Use(ProteccionCSRF())
	responderOK := func(c *gin.Context) { c.Status(http.StatusOK) }
	r.GET("/accion", responderOK)
	r.POST("/accion", responderOK)

	casos := []struct {
		nombre         string
		metodo         string
		encabezados    map[string]string
		codigoEsperado int
	}{
		{"cliente que no es navegador", http.MethodPost, nil, http.StatusOK},
		{"formulario del mismo sitio", http.MethodPost,
			map[string]string{"Sec-Fetch-Site": "same-origin"}, http.StatusOK},
		{"formulario desde otro sitio", http.MethodPost,
			map[string]string{"Sec-Fetch-Site": "cross-site"}, http.StatusForbidden},
		{"origen distinto en navegador antiguo", http.MethodPost,
			map[string]string{"Origin": "http://sitio-malicioso.com"}, http.StatusForbidden},
		{"lectura desde otro sitio", http.MethodGet,
			map[string]string{"Sec-Fetch-Site": "cross-site"}, http.StatusOK},
	}

	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			peticion := httptest.NewRequest(caso.metodo, "/accion", nil)
			for clave, valor := range caso.encabezados {
				peticion.Header.Set(clave, valor)
			}
			respuesta := httptest.NewRecorder()
			r.ServeHTTP(respuesta, peticion)

			if respuesta.Code != caso.codigoEsperado {
				t.Fatalf("código esperado %d, obtenido %d", caso.codigoEsperado, respuesta.Code)
			}
		})
	}
}
