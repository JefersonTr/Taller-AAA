package handlers

import (
	"net/http"

	"automarket/servicios"

	"github.com/gin-gonic/gin"
)

// ListarPendientes devuelve las publicaciones que esperan aprobación.
func ListarPendientes(c *gin.Context) {
	c.JSON(http.StatusOK, servicios.ListarPendientes())
}

// AutorizarPublicacion cambia una publicación pendiente a publicada,
// lo que la hace visible en el catálogo (HU-ADM-01).
func AutorizarPublicacion(c *gin.Context) {
	id, ok := leerID(c)
	if !ok {
		return
	}

	vehiculo, err := servicios.AutorizarPublicacion(id)
	if err != nil {
		responderErrorServicio(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"mensaje":  "publicación autorizada; ya es visible en el catálogo",
		"vehiculo": vehiculo,
	})
}

// EliminarPublicacion elimina una publicación en cualquier estado (HU-ADM-01).
func EliminarPublicacion(c *gin.Context) {
	id, ok := leerID(c)
	if !ok {
		return
	}

	if err := servicios.EliminarPublicacion(id); err != nil {
		responderErrorServicio(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"mensaje": "publicación eliminada"})
}

// EliminarUsuario elimina a un vendedor junto con sus publicaciones (HU-ADM-01).
func EliminarUsuario(c *gin.Context) {
	id, ok := leerID(c)
	if !ok {
		return
	}

	if err := servicios.EliminarUsuario(id); err != nil {
		responderErrorServicio(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"mensaje": "usuario eliminado junto con sus publicaciones"})
}
