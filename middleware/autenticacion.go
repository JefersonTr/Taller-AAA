package middleware

import (
	"errors"
	"net/http"

	"automarket/auth"
	"automarket/models"

	"github.com/gin-gonic/gin"
)

// claveUsuario es el nombre con el que se guarda el usuario autenticado
// dentro del contexto de Gin. Es privada (minúscula) para que solo este
// paquete pueda escribirla; los demás la leen con UsuarioActual.
const claveUsuario = "usuarioAutenticado"

// Autenticador describe lo único que el middleware necesita: verificar
// credenciales. El servicio del paquete auth la cumple, y en las pruebas
// usamos uno falso.
type Autenticador interface {
	Autenticar(nombreUsuario, contrasena string) (models.Usuario, error)
}

// RequiereAutenticacion protege una ruta: solo deja pasar peticiones con
// credenciales válidas enviadas por Basic Auth.
//
// Es la "primera A" del marco AAA: Authentication (¿quién eres?).
func RequiereAutenticacion(autenticador Autenticador) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Paso 1: leer usuario y contraseña del header Authorization.
		// "ok" es false si el header no existe o está mal formado.
		nombreUsuario, contrasena, ok := c.Request.BasicAuth()
		if !ok {
			rechazarNoAutenticado(c)
			return
		}

		// Paso 2: verificar las credenciales con el servicio.
		usuario, err := autenticador.Autenticar(nombreUsuario, contrasena)
		if errors.Is(err, auth.ErrCredencialesInvalidas) {
			rechazarNoAutenticado(c)
			return
		}
		if err != nil {
			// Falla del sistema, no del usuario: no damos detalles al cliente.
			c.AbortWithStatusJSON(http.StatusInternalServerError,
				gin.H{"error": "error interno del servidor"})
			return
		}

		// Paso 3: guardar el usuario para los siguientes eslabones de la cadena
		// (autorización, handler y Accounting).
		c.Set(claveUsuario, usuario)
		c.Next()
	}
}

// rechazarNoAutenticado corta la cadena y responde 401.
//
// El estándar HTTP exige que toda respuesta 401 incluya el header
// WWW-Authenticate, que le indica al cliente qué método de autenticación
// usar (aquí, Basic). Por eso un navegador muestra una ventana de login.
func rechazarNoAutenticado(c *gin.Context) {
	c.Header("WWW-Authenticate", `Basic realm="AutoMarket"`)
	c.AbortWithStatusJSON(http.StatusUnauthorized,
		gin.H{"error": "autenticación requerida"})
}

// UsuarioActual devuelve el usuario autenticado de la petición.
//
// El segundo valor es false si la petición no pasó por RequiereAutenticacion
// (por ejemplo, un visitante en una ruta pública).
func UsuarioActual(c *gin.Context) (models.Usuario, bool) {
	valor, existe := c.Get(claveUsuario)
	if !existe {
		return models.Usuario{}, false
	}

	// c.Get devuelve un valor de tipo "any" (cualquier tipo).
	// valor.(models.Usuario) comprueba que realmente sea un Usuario;
	// esto se llama "aserción de tipo".
	usuario, esUsuario := valor.(models.Usuario)
	return usuario, esUsuario
}
