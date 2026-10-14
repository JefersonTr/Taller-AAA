package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"automarket/auth"
	"automarket/models"

	"github.com/gin-gonic/gin"
)

// autenticadorFalso acepta únicamente a "ana" con "secreta123".
// Así probamos el middleware sin depender del hash ni del repositorio.
type autenticadorFalso struct{}

func (autenticadorFalso) Autenticar(nombreUsuario, contrasena string) (models.Usuario, error) {
	if nombreUsuario == "ana" && contrasena == "secreta123" {
		return models.Usuario{ID: 1, NombreUsuario: "ana", Rol: models.RolVendedor}, nil
	}
	return models.Usuario{}, auth.ErrCredencialesInvalidas
}

func TestRequiereAutenticacion(t *testing.T) {
	// Modo de prueba: Gin no imprime mensajes de depuración.
	gin.SetMode(gin.TestMode)

	// Ruta de prueba protegida: si deja pasar, responde el nombre del usuario.
	r := gin.New()
	r.GET("/protegida", RequiereAutenticacion(autenticadorFalso{}), func(c *gin.Context) {
		usuario, _ := UsuarioActual(c)
		c.String(http.StatusOK, usuario.NombreUsuario)
	})

	// Tabla de casos: cada fila es un escenario con su resultado esperado.
	casos := []struct {
		nombre          string
		conCredenciales bool
		usuario         string
		contrasena      string
		codigoEsperado  int
	}{
		{"sin credenciales", false, "", "", http.StatusUnauthorized},
		{"contraseña incorrecta", true, "ana", "otra", http.StatusUnauthorized},
		{"credenciales correctas", true, "ana", "secreta123", http.StatusOK},
	}

	for _, caso := range casos {
		// t.Run ejecuta cada caso como una subprueba con su propio nombre.
		t.Run(caso.nombre, func(t *testing.T) {
			// Crear la petición y, si corresponde, agregar Basic Auth.
			peticion := httptest.NewRequest(http.MethodGet, "/protegida", nil)
			if caso.conCredenciales {
				peticion.SetBasicAuth(caso.usuario, caso.contrasena)
			}

			// Ejecutar la petición y capturar la respuesta.
			respuesta := httptest.NewRecorder()
			r.ServeHTTP(respuesta, peticion)

			// Verificar el código HTTP.
			if respuesta.Code != caso.codigoEsperado {
				t.Fatalf("código esperado %d, obtenido %d", caso.codigoEsperado, respuesta.Code)
			}

			// Un 401 debe traer el header WWW-Authenticate.
			if respuesta.Code == http.StatusUnauthorized &&
				respuesta.Header().Get("WWW-Authenticate") == "" {
				t.Fatal("una respuesta 401 debe incluir el header WWW-Authenticate")
			}

			// Si pasó, el handler debe haber recibido al usuario correcto.
			if respuesta.Code == http.StatusOK && respuesta.Body.String() != "ana" {
				t.Fatalf("el handler recibió el usuario %q, se esperaba \"ana\"", respuesta.Body.String())
			}
		})
	}
}
