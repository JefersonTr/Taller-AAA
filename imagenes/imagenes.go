// Paquete imagenes: valida, limpia y guarda las fotos de los vehículos.
//
// No sabe nada de vehículos, usuarios ni HTTP: recibe los bytes de un
// archivo y devuelve una imagen JPEG segura, o un error que explica por qué
// se rechazó. Usa solo la librería estándar de Go.
package imagenes

import (
	"bytes"
	"crypto/rand"
	"errors"
	"image"
	"image/jpeg"
	_ "image/png" // Registra el lector de PNG para que image.Decode lo reconozca.
	"io/fs"
	"log"
	"net/http"
	"os"
	"path/filepath"
)

// Límites de las fotos.
const (
	// MaximoFotos es la cantidad máxima de fotos por publicación.
	MaximoFotos = 10

	// TamanoMaximo es el tamaño máximo de cada foto: 5 MB.
	// 5 << 20 significa 5 × 1.048.576 bytes (un "mega" en informática).
	TamanoMaximo = 5 << 20

	// PixelesMaximos limita ancho × alto (40 megapíxeles, más que cualquier
	// celular). Protege contra las "bombas de descompresión".
	PixelesMaximos = 40_000_000

	// calidadJPEG va de 1 a 100. 85 ofrece buena calidad con buen tamaño.
	calidadJPEG = 85
)

// Motivos por los que una foto se rechaza. El servicio los incluye en el
// mensaje que ve el vendedor.
var (
	ErrDemasiadoGrande = errors.New("supera el tamaño máximo de 5 MB")
	ErrFormato         = errors.New("no es una imagen JPEG o PNG")
	ErrDimensiones     = errors.New("tiene dimensiones demasiado grandes")
	ErrIlegible        = errors.New("está dañada o no se pudo leer")
)

// directorio es la carpeta donde se guardan las fotos. La define Configurar.
var directorio string

// Configurar indica en qué carpeta se guardan las fotos y la crea si no
// existe. Se llama una vez al arrancar el servidor.
func Configurar(ruta string) error {
	// 0o700: solo el dueño puede leer, escribir y entrar (en Linux/macOS).
	if err := os.MkdirAll(ruta, 0o700); err != nil {
		return err
	}
	directorio = ruta
	return nil
}

// Procesar valida una foto y la devuelve recodificada como JPEG.
//
// Las defensas se aplican de la más barata a la más costosa, y la imagen
// completa solo se abre cuando ya se sabe que es segura abrirla.
func Procesar(contenido []byte) ([]byte, error) {
	// Defensa 1: tamaño. No requiere mirar el contenido.
	if len(contenido) > TamanoMaximo {
		return nil, ErrDemasiadoGrande
	}

	// Defensa 2: tipo REAL. DetectContentType mira los primeros bytes del
	// archivo (su "firma"), no la extensión ni lo que diga el navegador.
	// Un programa renombrado como "carro.jpg" no tiene la firma de un JPEG.
	tipo := http.DetectContentType(contenido)
	if tipo != "image/jpeg" && tipo != "image/png" {
		return nil, ErrFormato
	}

	// Defensa 3: dimensiones. DecodeConfig lee SOLO la cabecera, donde
	// están el ancho y el alto, sin abrir la imagen completa.
	configuracion, _, err := image.DecodeConfig(bytes.NewReader(contenido))
	if err != nil {
		return nil, ErrIlegible
	}
	ancho, alto := configuracion.Width, configuracion.Height
	if ancho <= 0 || alto <= 0 || ancho*alto > PixelesMaximos {
		return nil, ErrDimensiones
	}

	// Defensa 4: abrir la imagen completa. Ya es seguro hacerlo.
	imagen, _, err := image.Decode(bytes.NewReader(contenido))
	if err != nil {
		return nil, ErrIlegible
	}

	// Defensa 5: recodificar. El JPEG nuevo contiene SOLO los píxeles. Se
	// pierden los metadatos EXIF (que pueden incluir la ubicación GPS donde
	// se tomó la foto) y cualquier contenido escondido en el archivo original.
	var salida bytes.Buffer
	if err := jpeg.Encode(&salida, imagen, &jpeg.Options{Quality: calidadJPEG}); err != nil {
		return nil, err
	}
	return salida.Bytes(), nil
}

// Guardar escribe las fotos en la carpeta configurada con nombres
// aleatorios, y devuelve esos nombres.
//
// Es "todo o nada": si alguna escritura falla, borra las que ya se habían
// escrito y devuelve el error.
func Guardar(fotos [][]byte) ([]string, error) {
	nombres := make([]string, 0, len(fotos))
	for _, foto := range fotos {
		if directorio == "" {
			return nil, errors.New("la carpeta de imágenes no está configurada")
		}

		// El nombre NUNCA viene del usuario: se genera al azar con
		// crypto/rand (26 caracteres). Así nadie puede usar un nombre como
		// "../../main.go" para escribir fuera de la carpeta, ni adivinar
		// los nombres de las fotos de otros.
		nombre := rand.Text() + ".jpg"

		// 0o600: solo el dueño (el servidor) puede leer y escribir el archivo.
		if err := os.WriteFile(filepath.Join(directorio, nombre), foto, 0o600); err != nil {
			Eliminar(nombres)
			return nil, err
		}
		nombres = append(nombres, nombre)
	}
	return nombres, nil
}

// Eliminar borra las fotos indicadas. Si alguna ya no existe, la ignora;
// cualquier otro problema queda registrado en el log del servidor.
func Eliminar(nombres []string) {
	if directorio == "" {
		return
	}
	for _, nombre := range nombres {
		// filepath.Base se queda solo con el nombre del archivo, por si
		// alguna vez llegara una ruta: nunca se borra fuera de la carpeta.
		ruta := filepath.Join(directorio, filepath.Base(nombre))
		if err := os.Remove(ruta); err != nil && !errors.Is(err, fs.ErrNotExist) {
			log.Printf("no se pudo eliminar la imagen %s: %v", nombre, err)
		}
	}
}
