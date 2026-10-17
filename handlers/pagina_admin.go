package handlers

import (
	"net/http"

	"automarket/servicios"

	"github.com/gin-gonic/gin"
)

// PaginaModeracion muestra el panel del administrador (HU-ADM-01).
func PaginaModeracion(c *gin.Context) {
	panel, err := servicios.ObtenerPanelModeracion()
	if err != nil {
		mostrarErrorServicio(c, err)
		return
	}

	mostrarPagina(c, http.StatusOK, "admin.html", gin.H{
		"Titulo": "Moderación",
		"Panel":  panel,
	})
}

// EnviarAutorizacion autoriza una publicación pendiente (HU-ADM-01).
func EnviarAutorizacion(c *gin.Context) {
	ejecutarAccionAdmin(c, func(id int) error {
		_, err := servicios.AutorizarPublicacion(id)
		return err
	}, "autorizado")
}

// EnviarEliminacionVehiculo elimina una publicación (HU-ADM-01).
func EnviarEliminacionVehiculo(c *gin.Context) {
	ejecutarAccionAdmin(c, servicios.EliminarPublicacion, "publicacion-eliminada")
}

// EnviarEliminacionUsuario elimina a un vendedor y sus publicaciones (HU-ADM-01).
func EnviarEliminacionUsuario(c *gin.Context) {
	ejecutarAccionAdmin(c, servicios.EliminarUsuario, "usuario-eliminado")
}

// ejecutarAccionAdmin aplica una acción del administrador sobre el ID de
// la ruta. Si sale bien, vuelve al panel con un aviso (Post/Redirect/Get);
// si falla, muestra la página de error que corresponda.
//
// "accion" es una FUNCIÓN recibida como parámetro: así los tres botones
// comparten este código y solo cambia lo que hacen.
func ejecutarAccionAdmin(c *gin.Context, accion func(id int) error, aviso string) {
	id, ok := idDeRuta(c)
	if !ok {
		mostrarError(c, http.StatusNotFound, "recurso no encontrado")
		return
	}

	if err := accion(id); err != nil {
		mostrarErrorServicio(c, err)
		return
	}

	c.Redirect(http.StatusSeeOther, "/admin?aviso="+aviso)
}
