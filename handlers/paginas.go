package handlers

import (
	"html/template"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

// ConfigurarPlantillas registra las funciones propias de las plantillas y
// carga los archivos HTML que coinciden con el patrón.
//
// Las funciones se registran ANTES de cargar las plantillas, porque al
// leerlas Go necesita saber que "precio" existe.
// Si el patrón no encuentra archivos, el programa se detiene al arrancar.
func ConfigurarPlantillas(r *gin.Engine, patron string) {
	r.SetFuncMap(template.FuncMap{
		"precio": FormatearPrecio,
	})
	r.LoadHTMLGlob(patron)
}

// FormatearPrecio convierte un precio en pesos al formato colombiano,
// con punto como separador de miles. Ejemplo: 78000000 → "$78.000.000".
func FormatearPrecio(precio int) string {
	texto := strconv.Itoa(precio)

	// strings.Builder arma un texto pieza por pieza de forma eficiente.
	var resultado strings.Builder
	for i, digito := range texto {
		// Se pone un punto antes de cada grupo de tres dígitos contados
		// desde la derecha (pero nunca al inicio).
		if i > 0 && (len(texto)-i)%3 == 0 {
			resultado.WriteByte('.')
		}
		resultado.WriteRune(digito)
	}
	return "$" + resultado.String()
}

// mostrarError muestra la página de error con el código HTTP indicado.
func mostrarError(c *gin.Context, codigo int, mensaje string) {
	c.HTML(codigo, "error.html", gin.H{
		"Titulo":  "Error",
		"Codigo":  codigo,
		"Mensaje": mensaje,
	})
}

// mostrarErrorServicio muestra como página el error de un servicio,
// usando la misma traducción de errores que la API.
func mostrarErrorServicio(c *gin.Context, err error) {
	codigo, mensaje := traducirError(err)
	mostrarError(c, codigo, mensaje)
}
