package handlers

import (
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"strconv"

	"automarket/imagenes"
	"automarket/models"
	"automarket/servicios"

	"github.com/gin-gonic/gin"
)

// limiteFormulario es el tamaño máximo de toda la petición de publicación:
// todas las fotos permitidas más 1 MB para los demás campos (51 MB).
const limiteFormulario = imagenes.MaximoFotos*imagenes.TamanoMaximo + 1<<20

// PaginaVendedor muestra las publicaciones del vendedor autenticado y el
// formulario para publicar un vehículo (HU-VEN-01).
func PaginaVendedor(c *gin.Context) {
	vendedor, ok := leerUsuarioDePagina(c)
	if !ok {
		return
	}
	mostrarPanelVendedor(c, http.StatusOK, vendedor, servicios.DatosPublicacion{}, "")
}

// EnviarPublicacion recibe el formulario para publicar un vehículo con sus
// fotos (HU-VEN-01). Si sale bien, redirige a "Mis vehículos" con un aviso
// (Post/Redirect/Get). Si falla, vuelve a mostrar la página con el error.
func EnviarPublicacion(c *gin.Context) {
	vendedor, ok := leerUsuarioDePagina(c)
	if !ok {
		return
	}

	// Barrera 1: si la petición DECLARA un tamaño mayor al permitido,
	// se rechaza sin leer ni un byte. 413 = "contenido demasiado grande".
	if c.Request.ContentLength > limiteFormulario {
		mostrarPanelVendedor(c, http.StatusRequestEntityTooLarge, vendedor, servicios.DatosPublicacion{},
			"las fotos superan el tamaño total permitido")
		return
	}

	// Barrera 2: si alguien miente en el tamaño declarado, MaxBytesReader
	// corta la lectura al llegar al límite.
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, limiteFormulario)

	fotos, err := leerFotos(c)
	if err != nil {
		mostrarPanelVendedor(c, http.StatusBadRequest, vendedor, servicios.DatosPublicacion{},
			"no se pudieron leer las fotos enviadas")
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
		Fotos:  fotos,
	}

	if _, err := servicios.PublicarVehiculo(vendedor.ID, datos); err != nil {
		codigo, mensaje := traducirError(err)
		datos.Fotos = nil // Las fotos no se devuelven a la página.
		mostrarPanelVendedor(c, codigo, vendedor, datos, mensaje)
		return
	}

	c.Redirect(http.StatusSeeOther, "/vendedor?aviso=publicado")
}

// leerFotos lee el contenido de los archivos enviados en el campo "fotos".
// Si el formulario no trae archivos (no es multipart), devuelve una lista vacía.
func leerFotos(c *gin.Context) ([][]byte, error) {
	formulario, err := c.MultipartForm()
	if errors.Is(err, http.ErrNotMultipart) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	archivos := formulario.File["fotos"]
	fotos := make([][]byte, 0, len(archivos))
	for _, archivo := range archivos {
		// Si el vendedor no eligió fotos, algunos navegadores envían un
		// archivo vacío. Se ignora.
		if archivo.Size == 0 {
			continue
		}
		contenido, err := leerArchivo(archivo)
		if err != nil {
			return nil, err
		}
		fotos = append(fotos, contenido)
	}
	return fotos, nil
}

// leerArchivo devuelve todos los bytes de un archivo recibido en el formulario.
func leerArchivo(archivo *multipart.FileHeader) ([]byte, error) {
	abierto, err := archivo.Open()
	if err != nil {
		return nil, err
	}
	defer abierto.Close()
	return io.ReadAll(abierto)
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
		"Titulo":      "Mis vehículos",
		"Vendedor":    vendedor,
		"Vehiculos":   servicios.ListarVehiculosDeVendedor(vendedor.ID),
		"Datos":       datos,
		"Error":       mensajeError,
		"MaximoFotos": imagenes.MaximoFotos,
	})
}
