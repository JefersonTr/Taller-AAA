package servicios

import (
	"bytes"
	"errors"
	"image"
	"image/png"
	"os"
	"testing"

	"automarket/imagenes"
	"automarket/models"
	"automarket/repository"
)

// fotoDePrueba devuelve los bytes de un PNG válido de 10 × 10 píxeles.
func fotoDePrueba(t *testing.T) []byte {
	t.Helper()
	var b bytes.Buffer
	if err := png.Encode(&b, image.NewRGBA(image.Rect(0, 0, 10, 10))); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}

// prepararFotos crea una base de datos limpia, configura una carpeta
// temporal para las fotos y devuelve esa carpeta.
func prepararFotos(t *testing.T) string {
	t.Helper()
	repository.InicializarBD()
	carpeta := t.TempDir()
	if err := imagenes.Configurar(carpeta); err != nil {
		t.Fatal(err)
	}
	return carpeta
}

// cantidadDeArchivos cuenta los archivos de una carpeta.
func cantidadDeArchivos(t *testing.T, carpeta string) int {
	t.Helper()
	archivos, err := os.ReadDir(carpeta)
	if err != nil {
		t.Fatal(err)
	}
	return len(archivos)
}

func TestPublicarConFotos(t *testing.T) {
	carpeta := prepararFotos(t)
	datos := DatosPublicacion{Marca: "Renault", Modelo: "Koleos 2018", Precio: 78000000}

	// Caso 1: dos fotos válidas quedan guardadas y asociadas al vehículo.
	datos.Fotos = [][]byte{fotoDePrueba(t), fotoDePrueba(t)}
	vehiculo, err := PublicarVehiculo(1, datos)
	if err != nil || len(vehiculo.Imagenes) != 2 {
		t.Fatal("debería publicar con dos fotos:", err)
	}
	if cantidadDeArchivos(t, carpeta) != 2 {
		t.Fatal("deberían existir dos archivos")
	}

	// Caso 2: todo o nada. Una foto válida y una falsa: se rechaza todo,
	// no se crea el vehículo y no queda ningún archivo nuevo.
	datos.Fotos = [][]byte{fotoDePrueba(t), []byte("no soy una foto")}
	if _, err := PublicarVehiculo(1, datos); !errors.Is(err, ErrDatosInvalidos) {
		t.Fatal("debería rechazar la publicación si una foto es inválida")
	}
	if cantidadDeArchivos(t, carpeta) != 2 {
		t.Fatal("una publicación rechazada no debe dejar archivos")
	}
	if len(repository.DB.ListarVehiculosPorVendedor(1)) != 1 {
		t.Fatal("una publicación rechazada no debe crear el vehículo")
	}

	// Caso 3: más del máximo permitido.
	datos.Fotos = make([][]byte, imagenes.MaximoFotos+1)
	if _, err := PublicarVehiculo(1, datos); !errors.Is(err, ErrDatosInvalidos) {
		t.Fatal("debería rechazar más fotos que el máximo")
	}

	// Caso 4: al eliminar la publicación, sus fotos se borran del disco.
	if err := EliminarPublicacion(vehiculo.ID); err != nil {
		t.Fatal(err)
	}
	if cantidadDeArchivos(t, carpeta) != 0 {
		t.Fatal("eliminar la publicación debería borrar sus fotos")
	}
}

func TestEliminarVendedorBorraSusFotos(t *testing.T) {
	carpeta := prepararFotos(t)

	ana, _ := repository.DB.GuardarUsuario(models.Usuario{NombreUsuario: "ana", Rol: models.RolVendedor})
	datos := DatosPublicacion{
		Marca: "Mazda", Modelo: "3 2020", Precio: 65000000,
		Fotos: [][]byte{fotoDePrueba(t)},
	}
	if _, err := PublicarVehiculo(ana.ID, datos); err != nil {
		t.Fatal(err)
	}

	if err := EliminarUsuario(ana.ID); err != nil {
		t.Fatal(err)
	}
	if cantidadDeArchivos(t, carpeta) != 0 {
		t.Fatal("eliminar al vendedor debería borrar las fotos de sus publicaciones")
	}
}
