package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"automarket/models"

	"github.com/gin-gonic/gin"
)

func TestRequiereRol(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// simularUsuario es un middleware de prueba que guarda un usuario con el
	// rol indicado, como si la petición ya hubiera pasado por
	// RequiereAutenticacion. Así probamos la autorización por separado.
	simularUsuario := func(rol models.Rol) gin.HandlerFunc {
		return func(c *gin.Context) {
			c.Set(claveUsuario, models.Usuario{ID: 1, Rol: rol})
			c.Next()
		}
	}

	// responderOK representa el handler final: si se ejecuta, responde 200.
	responderOK := func(c *gin.Context) {
		c.Status(http.StatusOK)
	}

	// Tres rutas que exigen rol administrador, con distintos usuarios.
	r := gin.New()
	r.GET("/como-admin", simularUsuario(models.RolAdministrador),
		RequiereRol(models.RolAdministrador), responderOK)
	r.GET("/como-vendedor", simularUsuario(models.RolVendedor),
		RequiereRol(models.RolAdministrador), responderOK)
	r.GET("/sin-usuario", RequiereRol(models.RolAdministrador), responderOK)

	casos := []struct {
		nombre         string
		ruta           string
		codigoEsperado int
	}{
		{"rol permitido", "/como-admin", http.StatusOK},
		{"rol no permitido", "/como-vendedor", http.StatusForbidden},
		{"sin autenticación previa", "/sin-usuario", http.StatusUnauthorized},
	}

	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			peticion := httptest.NewRequest(http.MethodGet, caso.ruta, nil)
			respuesta := httptest.NewRecorder()
			r.ServeHTTP(respuesta, peticion)

			if respuesta.Code != caso.codigoEsperado {
				t.Fatalf("código esperado %d, obtenido %d", caso.codigoEsperado, respuesta.Code)
			}
		})
	}
}
