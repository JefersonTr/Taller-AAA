package imagenes

import (
	"bytes"
	"encoding/binary"
	"hash/crc32"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"net/http"
	"os"
	"path/filepath"
	"testing"
)

// imagenDePrueba crea una imagen de 20 × 10 píxeles de un solo color.
func imagenDePrueba() image.Image {
	imagen := image.NewRGBA(image.Rect(0, 0, 20, 10))
	for x := 0; x < 20; x++ {
		for y := 0; y < 10; y++ {
			imagen.Set(x, y, color.RGBA{R: 200, G: 30, B: 30, A: 255})
		}
	}
	return imagen
}

// pngDePrueba devuelve los bytes de un PNG válido.
func pngDePrueba(t *testing.T) []byte {
	t.Helper()
	var b bytes.Buffer
	if err := png.Encode(&b, imagenDePrueba()); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}

// jpegConGPS devuelve un JPEG con un bloque de metadatos (APP1/EXIF) que
// contiene un texto que simula una ubicación GPS.
func jpegConGPS(t *testing.T) []byte {
	t.Helper()
	var original bytes.Buffer
	if err := jpeg.Encode(&original, imagenDePrueba(), nil); err != nil {
		t.Fatal(err)
	}
	// Un JPEG empieza con FF D8. Justo después se inserta el bloque
	// FF E1 (APP1), que es donde las cámaras guardan los datos EXIF.
	exif := []byte("Exif\x00\x00UBICACION-GPS-SECRETA")
	bloque := []byte{0xFF, 0xE1, byte((len(exif) + 2) >> 8), byte(len(exif) + 2)}
	bloque = append(bloque, exif...)

	resultado := append([]byte{0xFF, 0xD8}, bloque...)
	return append(resultado, original.Bytes()[2:]...)
}

// bombaPNG devuelve solo la cabecera de un PNG que dice medir
// ancho × alto. Pesa unos pocos bytes, pero abrirlo completo exigiría
// gigas de memoria: es una bomba de descompresión.
func bombaPNG(ancho, alto uint32) []byte {
	var b bytes.Buffer
	b.WriteString("\x89PNG\r\n\x1a\n") // Firma de todo archivo PNG

	// Bloque IHDR: ancho, alto, 8 bits por canal, color RGB.
	datos := make([]byte, 13)
	binary.BigEndian.PutUint32(datos[0:4], ancho)
	binary.BigEndian.PutUint32(datos[4:8], alto)
	datos[8] = 8
	datos[9] = 2

	tipoYDatos := append([]byte("IHDR"), datos...)
	binary.Write(&b, binary.BigEndian, uint32(len(datos)))
	b.Write(tipoYDatos)
	binary.Write(&b, binary.BigEndian, crc32.ChecksumIEEE(tipoYDatos))
	return b.Bytes()
}

func TestProcesar(t *testing.T) {
	// Caso 1: un PNG válido se convierte en JPEG.
	salida, err := Procesar(pngDePrueba(t))
	if err != nil {
		t.Fatal("un PNG válido debería aceptarse:", err)
	}
	if http.DetectContentType(salida) != "image/jpeg" {
		t.Fatal("la salida debería ser siempre JPEG")
	}

	// Caso 2: los metadatos GPS desaparecen al recodificar.
	salida, err = Procesar(jpegConGPS(t))
	if err != nil {
		t.Fatal("un JPEG con metadatos debería aceptarse:", err)
	}
	if bytes.Contains(salida, []byte("UBICACION-GPS-SECRETA")) {
		t.Fatal("los metadatos GPS deberían eliminarse")
	}

	// Casos de rechazo: cada defensa con su error.
	rechazos := []struct {
		nombre      string
		contenido   []byte
		errEsperado error
	}{
		{"texto disfrazado de imagen", []byte("esto no es una foto, es un programa"), ErrFormato},
		{"archivo de más de 5 MB", bytes.Repeat([]byte{0}, TamanoMaximo+1), ErrDemasiadoGrande},
		{"bomba de descompresión", bombaPNG(50000, 50000), ErrDimensiones},
	}
	for _, caso := range rechazos {
		t.Run(caso.nombre, func(t *testing.T) {
			if _, err := Procesar(caso.contenido); err != caso.errEsperado {
				t.Fatalf("error esperado %q, obtenido %v", caso.errEsperado, err)
			}
		})
	}
}

func TestGuardarYEliminar(t *testing.T) {
	if err := Configurar(t.TempDir()); err != nil {
		t.Fatal(err)
	}

	nombres, err := Guardar([][]byte{[]byte("foto 1"), []byte("foto 2")})
	if err != nil || len(nombres) != 2 {
		t.Fatal("debería guardar las dos fotos:", err)
	}

	// Los nombres son aleatorios, distintos y terminan en .jpg.
	if nombres[0] == nombres[1] || filepath.Ext(nombres[0]) != ".jpg" {
		t.Fatalf("nombres inesperados: %v", nombres)
	}

	// Los archivos existen y, al eliminarlos, desaparecen.
	for _, nombre := range nombres {
		if _, err := os.Stat(filepath.Join(directorio, nombre)); err != nil {
			t.Fatalf("la foto %s debería existir", nombre)
		}
	}
	Eliminar(nombres)
	for _, nombre := range nombres {
		if _, err := os.Stat(filepath.Join(directorio, nombre)); err == nil {
			t.Fatalf("la foto %s debería haberse eliminado", nombre)
		}
	}
}
