package servicios

import (
	"fmt"
	"strings"

	"automarket/auth"
	"automarket/imagenes"
	"automarket/models"
	"automarket/repository"
)

// DatosRegistro contiene lo que una persona envía para registrarse como vendedor.
//
// No incluye rol ni ID: el cliente NO puede elegirlos. Esa es la protección
// contra la asignación masiva (registrarse como administrador).
type DatosRegistro struct {
	Nombre        string `json:"nombre"`
	NombreUsuario string `json:"nombre_usuario"`
	Telefono      string `json:"telefono"`
	Contrasena    string `json:"contrasena"`
}

// validar comprueba que los datos del registro sean aceptables.
func (d DatosRegistro) validar() error {
	// strings.TrimSpace quita los espacios de los extremos: así un campo
	// lleno solo de espacios cuenta como vacío.
	if strings.TrimSpace(d.Nombre) == "" || strings.TrimSpace(d.NombreUsuario) == "" ||
		strings.TrimSpace(d.Telefono) == "" || d.Contrasena == "" {
		return fmt.Errorf("%w: todos los campos son obligatorios", ErrDatosInvalidos)
	}
	// Basic Auth separa usuario y contraseña con ':'; un nombre de usuario
	// con ese carácter nunca podría autenticarse.
	if strings.Contains(d.NombreUsuario, ":") {
		return fmt.Errorf("%w: el nombre de usuario no puede contener ':'", ErrDatosInvalidos)
	}
	return nil
}

// DatosPublicacion contiene lo que un vendedor envía para publicar un vehículo.
//
// No incluye estado ni vendedor: los asigna el servidor.
type DatosPublicacion struct {
	Marca  string `json:"marca"`
	Modelo string `json:"modelo"`
	Precio int    `json:"precio"`

	// Fotos son los bytes de cada archivo, tal como llegaron. Solo se
	// envían desde el formulario web; json:"-" impide recibirlas por JSON.
	Fotos [][]byte `json:"-"`
}

// validar comprueba que los datos de la publicación sean aceptables.
func (d DatosPublicacion) validar() error {
	if strings.TrimSpace(d.Marca) == "" || strings.TrimSpace(d.Modelo) == "" {
		return fmt.Errorf("%w: la marca y el modelo son obligatorios", ErrDatosInvalidos)
	}
	if d.Precio <= 0 {
		return fmt.Errorf("%w: el precio debe ser mayor que cero", ErrDatosInvalidos)
	}
	if len(d.Fotos) > imagenes.MaximoFotos {
		return fmt.Errorf("%w: se permiten máximo %d fotos", ErrDatosInvalidos, imagenes.MaximoFotos)
	}
	return nil
}

// RegistrarVendedor crea la cuenta de un vendedor con su contraseña
// convertida en hash (HU-VEN-01).
//
// Errores: ErrDatosInvalidos; ErrNombreUsuarioEnUso si el nombre ya existe.
func RegistrarVendedor(datos DatosRegistro) (models.Usuario, error) {
	if err := datos.validar(); err != nil {
		return models.Usuario{}, err
	}

	hash, err := auth.GenerarHash(datos.Contrasena)
	if err != nil {
		return models.Usuario{}, err
	}

	// El rol lo asigna el servidor, nunca el cliente.
	return repository.DB.GuardarUsuario(models.Usuario{
		Nombre:         datos.Nombre,
		NombreUsuario:  datos.NombreUsuario,
		Telefono:       datos.Telefono,
		Rol:            models.RolVendedor,
		HashContrasena: hash,
	})
}

// PublicarVehiculo registra un vehículo en estado pendiente de aprobación,
// asociado al vendedor indicado, con sus fotos (HU-VEN-01).
//
// Es "todo o nada": si una sola foto es inválida, no se guarda nada.
//
// Errores: ErrDatosInvalidos (con el motivo y el número de la foto), o un
// error de guardado en disco.
func PublicarVehiculo(vendedorID int, datos DatosPublicacion) (models.Vehiculo, error) {
	// Paso 1: validar los datos y la cantidad de fotos.
	if err := datos.validar(); err != nil {
		return models.Vehiculo{}, err
	}

	// Paso 2: procesar TODAS las fotos antes de guardar nada. Si una falla,
	// todavía no se ha escrito ningún archivo.
	procesadas := make([][]byte, 0, len(datos.Fotos))
	for i, foto := range datos.Fotos {
		limpia, err := imagenes.Procesar(foto)
		if err != nil {
			return models.Vehiculo{}, fmt.Errorf("%w: la foto %d %v", ErrDatosInvalidos, i+1, err)
		}
		procesadas = append(procesadas, limpia)
	}

	// Paso 3: guardar los archivos. Si uno falla, Guardar borra los demás.
	nombres, err := imagenes.Guardar(procesadas)
	if err != nil {
		return models.Vehiculo{}, err
	}

	// Paso 4: guardar el vehículo. Si falla, se borran las fotos para no
	// dejar archivos que no pertenecen a ninguna publicación.
	vehiculo, err := repository.DB.GuardarVehiculo(models.Vehiculo{
		Marca:      datos.Marca,
		Modelo:     datos.Modelo,
		Precio:     datos.Precio,
		Estado:     models.EstadoPendiente,
		VendedorID: vendedorID,
		Imagenes:   nombres,
	})
	if err != nil {
		imagenes.Eliminar(nombres)
		return models.Vehiculo{}, err
	}
	return vehiculo, nil
}

// ReportarVenta marca como vendido un vehículo publicado del vendedor,
// lo que lo retira del catálogo (HU-VEN-01).
//
// Errores: ErrNoEncontrado; ErrSinPermiso si no es su dueño;
// ErrEstadoNoValido si no está publicado.
func ReportarVenta(vendedorID, vehiculoID int) (models.Vehiculo, error) {
	// Paso 1: buscar el vehículo.
	vehiculo, err := repository.DB.BuscarVehiculoPorID(vehiculoID)
	if err != nil {
		return models.Vehiculo{}, err
	}

	// Paso 2: autorización por propiedad. Va ANTES de revisar el estado
	// para no revelar información de vehículos ajenos.
	if vehiculo.VendedorID != vendedorID {
		return models.Vehiculo{}, ErrSinPermiso
	}

	// Paso 3: regla de negocio, solo se vende lo que está en el catálogo.
	if vehiculo.Estado != models.EstadoPublicada {
		return models.Vehiculo{}, ErrEstadoNoValido
	}

	return repository.DB.ActualizarEstadoVehiculo(vehiculoID, models.EstadoVendido)
}

// ListarVehiculosDeVendedor devuelve todas las publicaciones del vendedor,
// en cualquier estado. La usa la página "Mis vehículos".
func ListarVehiculosDeVendedor(vendedorID int) []models.Vehiculo {
	return repository.DB.ListarVehiculosPorVendedor(vendedorID)
}
