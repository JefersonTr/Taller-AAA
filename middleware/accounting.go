// Paquete middleware: funciones que se ejecutan antes y después de los handlers.
package middleware

import (
	"log"
	"time"

	"automarket/models"
	"automarket/repository"

	"github.com/gin-gonic/gin"
)

// Accounting registra cada operación que recibe la API (la "A" de Accounting en AAA).
//
// Debe ser el PRIMER middleware de la cadena: así envuelve a todos los demás
// y también queda registrado cuando una petición es rechazada (401, 403)
// o cuando ocurre un error inesperado (500).
func Accounting() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 1. Guardar el momento en que llegó la petición.
		inicio := time.Now().UTC()

		// 2. Pasar la petición al resto de la cadena (autenticación, handler...).
		//    El código de abajo se ejecuta cuando la respuesta ya está decidida.
		c.Next()

		// 3. Identificar quién hizo la operación. Si la ruta es pública o la
		//    autenticación falló, no hay usuario en el contexto: es "anonimo".
		usuario := "anonimo"
		if autenticado, ok := UsuarioActual(c); ok {
			usuario = autenticado.NombreUsuario
		}

		// 4. Armar y guardar el registro de la operación.
		registro := models.RegistroAuditoria{
			FechaHora:    inicio.Format(time.RFC3339),
			Metodo:       c.Request.Method,
			Ruta:         c.Request.URL.Path,
			IPCliente:    c.ClientIP(),
			Usuario:      usuario,
			CodigoEstado: c.Writer.Status(),
		}

		if err := repository.DB.GuardarRegistro(registro); err != nil {
			// La respuesta ya se envió al cliente; solo podemos dejar constancia.
			log.Printf("[ACCOUNTING] ERROR: no se pudo guardar el registro: %v", err)
			return
		}

		// 5. Mostrarlo también en consola.
		log.Printf("[ACCOUNTING] %s | %s %s | IP: %s | Usuario: %s | Estado: %d",
			registro.FechaHora, registro.Metodo, registro.Ruta,
			registro.IPCliente, registro.Usuario, registro.CodigoEstado)
	}
}
