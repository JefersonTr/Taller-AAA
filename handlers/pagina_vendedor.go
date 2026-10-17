package handlers

import (
	"net/http"
	"strconv"

	"automarket/models"
	"automarket/servicios"

	"github.com/gin-gonic/gin"
)

// PaginaVendedor muestra las publicaciones del vendedor autenticado y el
// formulario para publicar un vehículo (HU-VEN-01).
func PaginaVendedor(c *gin.Context) {
	vendedor, ok := leerUsuarioDePagina(c)
	if !ok {
		return
	}
	mostrarPanelVendedor(c, http.StatusOK, vendedor, servicios.DatosPublicacion{}, "")
}

// EnviarPublicacion recibe el formulario para publicar un vehículo (HU-VEN-01).
// Si sale bien, redirige a "Mis vehículos" con un aviso (Post/Redirect/Get).
// Si falla, vuelve a mostrar la página con el error y los datos escritos.
func EnviarPublicacion(c *gin.Context) {
	vendedor, ok := leerUsuarioDePagina(c)
	if !ok {
		return
	}

	// Si el precio no es un número, queda en 0 y el servicio lo rechaza
	// con el mensaje "el precio debe ser mayor que cero".
	precio, _ := strconv.Atoi(c.PostForm("precio"))

	// Se leen SOLO los campos permitidos: el estado y el vendedor los
	// asigna el servicio, nunca el formulario.
	datos := servicios.DatosPublicacion{
		Marca:  c.PostForm("marca"),
		Modelo: c.PostForm("modelo"),
		Precio: precio,
	}

	if _, err := servicios.PublicarVehiculo(vendedor.ID, datos); err != nil {
		codigo, mensaje := traducirError(err)
		mostrarPanelVendedor(c, codigo, vendedor, datos, mensaje)
		return
	}

	c.Redirect(http.StatusSeeOther, "/vendedor?aviso=publicado")
}

// EnviarVenta recibe el botón "Reportar venta" de una publicación (HU-VEN-01).
func EnviarVenta(c *gin.Context) {
	vendedor, ok := leerUsuarioDePagina(c)
	if !ok {
		return
	}

	id, ok := idDeRuta(c)
	if !ok {
		mostrarError(c, http.StatusNotFound, "vehículo no encontrado")
		return
	}

	// El servicio verifica que el vehículo sea del vendedor (403) y que
	// esté publicado (409), igual que en la API.
	if _, err := servicios.ReportarVenta(vendedor.ID, id); err != nil {
		mostrarErrorServicio(c, err)
		return
	}

	c.Redirect(http.StatusSeeOther, "/vendedor?aviso=vendido")
}

// mostrarPanelVendedor muestra la página "Mis vehículos". La usan los tres
// handlers para no repetir los datos que necesita la plantilla.
func mostrarPanelVendedor(c *gin.Context, codigo int, vendedor models.Usuario,
	datos servicios.DatosPublicacion, mensajeError string) {
	mostrarPagina(c, codigo, "vendedor.html", gin.H{
		"Titulo":    "Mis vehículos",
		"Vendedor":  vendedor,
		"Vehiculos": servicios.ListarVehiculosDeVendedor(vendedor.ID),
		"Datos":     datos,
		"Error":     mensajeError,
	})
}
