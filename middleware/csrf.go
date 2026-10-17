package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// ProteccionCSRF rechaza las peticiones que modifican datos (POST, PUT,
// DELETE...) cuando un navegador las envía desde OTRO sitio web.
//
// ¿Por qué hace falta? Con Basic Auth, el navegador recuerda el usuario y la
// contraseña y los envía solo en cada petición a AutoMarket. Si un vendedor
// visita una página maliciosa, esa página podría enviar un formulario a
// AutoMarket y el navegador agregaría las credenciales sin que el vendedor
// lo note. Eso es CSRF (falsificación de peticiones entre sitios).
//
// Usa http.CrossOriginProtection, de la librería estándar de Go
// que revisa los encabezados que agregan los navegadores:
//   - Sec-Fetch-Site: indica si la petición viene del mismo sitio o de otro.
//   - Origin: indica desde qué sitio se envió (para navegadores antiguos).
//
// Los métodos que solo leen (GET, HEAD, OPTIONS) siempre pasan. Los clientes
// que no son navegadores (PowerShell, las pruebas) no envían esos
// encabezados y también pasan: un ataque CSRF solo es posible a través del
// navegador de la víctima.
func ProteccionCSRF() gin.HandlerFunc {
	proteccion := http.NewCrossOriginProtection()

	return func(c *gin.Context) {
		if err := proteccion.Check(c.Request); err != nil {
			c.AbortWithStatusJSON(http.StatusForbidden,
				gin.H{"error": "solicitud rechazada: proviene de otro sitio"})
			return
		}
		c.Next()
	}
}
