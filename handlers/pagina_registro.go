package handlers

import (
	"net/http"

	"automarket/servicios"

	"github.com/gin-gonic/gin"
)

// PaginaRegistro muestra el formulario de registro de vendedores (HU-VEN-01).
func PaginaRegistro(c *gin.Context) {
	// "Datos" siempre debe existir (aunque vacío), porque la plantilla
	// lee .Datos.Nombre y fallaría si no hubiera nada.
	mostrarPagina(c, http.StatusOK, "registro.html", gin.H{
		"Titulo": "Crear cuenta",
		"Datos":  servicios.DatosRegistro{},
	})
}

// EnviarRegistro recibe el formulario de registro (HU-VEN-01).
//
// Si sale bien, redirige al catálogo con un aviso (patrón Post/Redirect/Get).
// Si falla, vuelve a mostrar el formulario con el error y los datos escritos.
func EnviarRegistro(c *gin.Context) {
	// Se leen SOLO los campos permitidos. Un campo extra como "rol"
	// en el formulario se ignora (protección contra asignación masiva).
	datos := servicios.DatosRegistro{
		Nombre:        c.PostForm("nombre"),
		NombreUsuario: c.PostForm("nombre_usuario"),
		Telefono:      c.PostForm("telefono"),
		Contrasena:    c.PostForm("contrasena"),
	}

	if _, err := servicios.RegistrarVendedor(datos); err != nil {
		codigo, mensaje := traducirError(err)
		datos.Contrasena = "" // La contraseña nunca se devuelve a la página.
		mostrarPagina(c, codigo, "registro.html", gin.H{
			"Titulo": "Crear cuenta",
			"Datos":  datos,
			"Error":  mensaje,
		})
		return
	}

	// 303 See Other: le indica al navegador que haga un GET a esta dirección.
	c.Redirect(http.StatusSeeOther, "/?aviso=registrado")
}
