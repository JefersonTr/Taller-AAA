package handlers

import (
	"html/template"
	"net/http"
	"strconv"
	"strings"

	"automarket/middleware"
	"automarket/models"

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

// avisos son los mensajes que se pueden mostrar después de una acción.
//
// La URL solo lleva un código (por ejemplo, /?aviso=registrado), nunca el
// texto. Así nadie puede fabricar un enlace que muestre un mensaje falso
// dentro del sitio. Un código que no esté aquí no muestra nada.
var avisos = map[string]string{
	"registrado":            "Tu cuenta fue creada. Entra a «Mis vehículos» con tu usuario y contraseña para publicar.",
	"publicado":             "Tu vehículo quedó pendiente de aprobación. Aparecerá en el catálogo cuando un administrador lo autorice.",
	"vendido":               "Venta reportada. El vehículo salió del catálogo.",
	"autorizado":            "Publicación autorizada. Ya es visible en el catálogo.",
	"publicacion-eliminada": "Publicación eliminada.",
	"usuario-eliminado":     "Vendedor eliminado junto con todas sus publicaciones.",
}

// mostrarPagina muestra una plantilla agregando a sus datos el aviso que
// corresponda al código "?aviso=" de la URL, si existe.
func mostrarPagina(c *gin.Context, codigo int, plantilla string, datos gin.H) {
	// Si el código no existe en el mapa, el resultado es "" y no se muestra nada.
	datos["Aviso"] = avisos[c.Query("aviso")]
	c.HTML(codigo, plantilla, datos)
}

// mostrarError muestra la página de error con el código HTTP indicado.
func mostrarError(c *gin.Context, codigo int, mensaje string) {
	mostrarPagina(c, codigo, "error.html", gin.H{
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

// leerUsuarioDePagina obtiene el usuario autenticado de la petición.
// Si no lo hay (una ruta configurada sin autenticación), muestra la página
// de error 401 y devuelve false ("fallar cerrado").
func leerUsuarioDePagina(c *gin.Context) (models.Usuario, bool) {
	usuario, autenticado := middleware.UsuarioActual(c)
	if !autenticado {
		mostrarError(c, http.StatusUnauthorized, "autenticación requerida")
	}
	return usuario, autenticado
}

// RutaNoEncontrada responde a las direcciones que no existen: la página de
// error para un navegador y JSON para un cliente de la API.
func RutaNoEncontrada(c *gin.Context) {
	middleware.ResponderError(c, http.StatusNotFound, "la página que buscas no existe")
}
