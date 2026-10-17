package servicios

import (
	"errors"
	"testing"

	"automarket/models"
	"automarket/repository"
)

// datosPrueba agrupa los datos que crea prepararDatos.
type datosPrueba struct {
	admin, vendedor      models.Usuario
	pendiente, publicado models.Vehiculo
}

// prepararDatos crea una base de datos limpia con un administrador,
// un vendedor y dos vehículos del vendedor: uno pendiente y uno publicado.
func prepararDatos(t *testing.T) datosPrueba {
	// t.Helper() hace que, si algo falla aquí, Go señale la línea de la
	// prueba que llamó a esta función, que es la que interesa.
	t.Helper()
	repository.InicializarBD()

	var d datosPrueba
	var err error
	if d.admin, err = repository.DB.GuardarUsuario(models.Usuario{
		NombreUsuario: "admin", Rol: models.RolAdministrador,
	}); err != nil {
		t.Fatal(err)
	}
	if d.vendedor, err = repository.DB.GuardarUsuario(models.Usuario{
		NombreUsuario: "ana", Rol: models.RolVendedor,
	}); err != nil {
		t.Fatal(err)
	}
	d.pendiente = repository.DB.GuardarVehiculo(models.Vehiculo{
		Marca: "Mazda", Modelo: "3 2020", Precio: 65000000,
		Estado: models.EstadoPendiente, VendedorID: d.vendedor.ID,
	})
	d.publicado = repository.DB.GuardarVehiculo(models.Vehiculo{
		Marca: "Renault", Modelo: "Koleos 2018", Precio: 78000000,
		Estado: models.EstadoPublicada, VendedorID: d.vendedor.ID,
	})
	return d
}

func TestListarPendientes(t *testing.T) {
	d := prepararDatos(t)

	pendientes := ListarPendientes()
	if len(pendientes) != 1 || pendientes[0].ID != d.pendiente.ID {
		t.Fatal("solo debe listar las publicaciones pendientes")
	}
}

func TestAutorizarPublicacion(t *testing.T) {
	d := prepararDatos(t)

	// Caso 1: una publicación pendiente pasa a publicada.
	autorizado, err := AutorizarPublicacion(d.pendiente.ID)
	if err != nil || autorizado.Estado != models.EstadoPublicada {
		t.Fatal("debería autorizar una publicación pendiente")
	}

	// Caso 2: una publicación que ya está publicada no se puede autorizar.
	if _, err := AutorizarPublicacion(d.publicado.ID); !errors.Is(err, ErrEstadoNoValido) {
		t.Fatal("debería devolver ErrEstadoNoValido si no está pendiente")
	}

	// Caso 3: una publicación inexistente.
	if _, err := AutorizarPublicacion(99); !errors.Is(err, ErrNoEncontrado) {
		t.Fatal("debería devolver ErrNoEncontrado si no existe")
	}
}

func TestEliminarPublicacion(t *testing.T) {
	d := prepararDatos(t)

	// Caso 1: eliminar una publicación existente.
	if err := EliminarPublicacion(d.publicado.ID); err != nil {
		t.Fatal("debería eliminar la publicación:", err)
	}

	// Caso 2: eliminarla otra vez ya no es posible.
	if err := EliminarPublicacion(d.publicado.ID); !errors.Is(err, ErrNoEncontrado) {
		t.Fatal("eliminar dos veces debería devolver ErrNoEncontrado")
	}
}

func TestEliminarUsuario(t *testing.T) {
	d := prepararDatos(t)

	// Caso 1: un administrador está protegido y sigue existiendo.
	if err := EliminarUsuario(d.admin.ID); !errors.Is(err, ErrAdministradorProtegido) {
		t.Fatal("no debería permitir eliminar a un administrador")
	}
	if _, err := repository.DB.BuscarUsuarioPorID(d.admin.ID); err != nil {
		t.Fatal("el administrador no debería haberse eliminado")
	}

	// Caso 2: un vendedor se elimina junto con sus publicaciones.
	if err := EliminarUsuario(d.vendedor.ID); err != nil {
		t.Fatal("debería eliminar al vendedor:", err)
	}
	if _, err := repository.DB.BuscarVehiculoPorID(d.pendiente.ID); !errors.Is(err, ErrNoEncontrado) {
		t.Fatal("las publicaciones del vendedor eliminado también deben eliminarse")
	}

	// Caso 3: un usuario inexistente.
	if err := EliminarUsuario(99); !errors.Is(err, ErrNoEncontrado) {
		t.Fatal("debería devolver ErrNoEncontrado si no existe")
	}
}

func TestObtenerPanelModeracion(t *testing.T) {
	d := prepararDatos(t)

	panel, err := ObtenerPanelModeracion()
	if err != nil {
		t.Fatal(err)
	}

	// Una publicación en cada lista, con el nombre de usuario del vendedor
	// disponible para mostrarlo.
	if len(panel.Pendientes) != 1 || panel.Pendientes[0].ID != d.pendiente.ID {
		t.Fatal("el panel debe listar la publicación pendiente")
	}
	if len(panel.Publicados) != 1 || panel.Publicados[0].ID != d.publicado.ID {
		t.Fatal("el panel debe listar la publicación del catálogo")
	}

	// Solo la vendedora; el administrador no aparece.
	if len(panel.Vendedores) != 1 || panel.Vendedores[0].ID != d.vendedor.ID {
		t.Fatal("el panel debe listar solo a los vendedores")
	}
}
