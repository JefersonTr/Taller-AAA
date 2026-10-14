package middleware

import (
	"net/http"

	"automarket/models"

	"github.com/gin-gonic/gin"
)

// RequiereRol solo deja pasar a usuarios autenticados con el rol indicado.
//
// Es la "segunda A" del marco AAA: Authorization (¿qué puedes hacer?).
// Debe ir DESPUÉS de RequiereAutenticacion, porque necesita saber
// quién es el usuario para revisar su rol.
func RequiereRol(rolPermitido models.Rol) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Paso 1: obtener el usuario que dejó RequiereAutenticacion.
		usuario, autenticado := UsuarioActual(c)
		if !autenticado {
			// Si llegamos aquí sin usuario, la ruta quedó mal configurada
			// (falta RequiereAutenticacion antes). Ante la duda, negamos
			// el acceso en lugar de dejar pasar: esto se llama "fallar cerrado".
			rechazarNoAutenticado(c)
			return
		}

		// Paso 2: comparar el rol del usuario con el rol que exige la ruta.
		if usuario.Rol != rolPermitido {
			c.AbortWithStatusJSON(http.StatusForbidden,
				gin.H{"error": "no tiene permisos para esta operación"})
			return
		}

		// Paso 3: el rol es correcto, la petición sigue al handler.
		c.Next()
	}
}
