package middleware

import (
	"github.com/gin-gonic/gin"
)

// ResponderError corta la cadena de middlewares y responde un error en el
// formato que prefiera quien hizo la petición:
//   - Un navegador, que pide "text/html" en su encabezado Accept, recibe la
//     página de error del sitio, con el mismo diseño de las demás páginas.
//   - Cualquier otro cliente (la API, PowerShell, las pruebas) recibe JSON.
//
// Esto se llama "negociación de contenido": el mismo error, entregado en el
// formato que cada cliente entiende.
func ResponderError(c *gin.Context, codigo int, mensaje string) {
	// JSON va PRIMERO en la lista: si el cliente no envía Accept, o acepta
	// cualquier formato (*/*), Gin elige el primero. Así la API no cambia.
	if c.NegotiateFormat(gin.MIMEJSON, gin.MIMEHTML) == gin.MIMEHTML {
		c.HTML(codigo, "error.html", gin.H{
			"Titulo":  "Error",
			"Codigo":  codigo,
			"Mensaje": mensaje,
		})
		c.Abort()
		return
	}
	c.AbortWithStatusJSON(codigo, gin.H{"error": mensaje})
}
