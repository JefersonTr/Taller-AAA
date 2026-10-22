package handlers

import (
	"bytes"
	"fmt"
	"image"
	"image/png"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"automarket/imagenes"
	"automarket/repository"

	"github.com/gin-gonic/gin"
)

// pngPequeno devuelve los bytes de un PNG válido de 10 × 10 píxeles.
func pngPequeno(t *testing.T) []byte {
	t.Helper()
	var b bytes.Buffer
	if err := png.Encode(&b, image.NewRGBA(image.Rect(0, 0, 10, 10))); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}

// formularioConFotos arma el cuerpo de un formulario con archivos, igual que
// un navegador con enctype="multipart/form-data". Devuelve el cuerpo y el
// Content-Type, que incluye el separador entre las partes.
func formularioConFotos(t *testing.T, campos map[string]string, fotos [][]byte) (io.Reader, string) {
	t.Helper()
	var cuerpo bytes.Buffer
	escritor := multipart.NewWriter(&cuerpo)

	for nombre, valor := range campos {
		if err := escritor.WriteField(nombre, valor); err != nil {
			t.Fatal(err)
		}
	}
	for i, foto := range fotos {
		parte, err := escritor.CreateFormFile("fotos", fmt.Sprintf("foto%d.png", i+1))
		if err != nil {
			t.Fatal(err)
		}
		if _, err := parte.Write(foto); err != nil {
			t.Fatal(err)
		}
	}
	if err := escritor.Close(); err != nil {
		t.Fatal(err)
	}
	return &cuerpo, escritor.FormDataContentType()
}

// publicarComoAna envía el formulario de publicación con las credenciales de Ana.
func publicarComoAna(r *gin.Engine, peticion *http.Request, tipo string) *httptest.ResponseRecorder {
	peticion.Header.Set("Content-Type", tipo)
	peticion.SetBasicAuth("ana", "clave")
	respuesta := httptest.NewRecorder()
	r.ServeHTTP(respuesta, peticion)
	return respuesta
}

func TestPublicarConFotosDesdeLaPagina(t *testing.T) {
	r := prepararPaginasVendedor(t)
	if err := imagenes.Configurar(t.TempDir()); err != nil {
		t.Fatal(err)
	}
	campos := map[string]string{"marca": "Chevrolet", "modelo": "Onix 2021", "precio": "55000000"}

	// Caso 1: dos fotos válidas. Redirige y el vehículo 4 queda con dos fotos.
	cuerpo, tipo := formularioConFotos(t, campos, [][]byte{pngPequeno(t), pngPequeno(t)})
	respuesta := publicarComoAna(r, httptest.NewRequest(http.MethodPost, "/vendedor/vehiculos", cuerpo), tipo)
	if respuesta.Code != http.StatusSeeOther {
		t.Fatalf("debería redirigir con 303; obtuvo %d: %s", respuesta.Code, respuesta.Body.String())
	}
	vehiculo, err := repository.DB.BuscarVehiculoPorID(4)
	if err != nil || len(vehiculo.Imagenes) != 2 {
		t.Fatal("el vehículo publicado debería tener dos fotos")
	}

	// Caso 2: una foto falsa. La página muestra cuál foto falló.
	cuerpo, tipo = formularioConFotos(t, campos, [][]byte{[]byte("no soy una foto")})
	respuesta = publicarComoAna(r, httptest.NewRequest(http.MethodPost, "/vendedor/vehiculos", cuerpo), tipo)
	if respuesta.Code != http.StatusBadRequest || !strings.Contains(respuesta.Body.String(), "la foto 1") {
		t.Fatalf("debería indicar que la foto 1 es inválida; obtuvo %d", respuesta.Code)
	}

	// Caso 3: una petición que declara más del tamaño permitido se rechaza
	// con 413 antes de leer su contenido.
	peticion := httptest.NewRequest(http.MethodPost, "/vendedor/vehiculos", strings.NewReader(""))
	peticion.ContentLength = limiteFormulario + 1
	respuesta = publicarComoAna(r, peticion, "multipart/form-data; boundary=x")
	if respuesta.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("debería responder 413; obtuvo %d", respuesta.Code)
	}
}
