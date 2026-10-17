package repository

import (
	"errors"
	"testing"

	"automarket/models"
)

func TestActualizarEstadoVehiculo(t *testing.T) {
	// Preparación: base de datos limpia con un vehículo publicado.
	InicializarBD()
	guardado, _ := DB.GuardarVehiculo(models.Vehiculo{
		Marca: "Renault", Modelo: "Koleos 2018", Precio: 78000000,
		Estado: models.EstadoPublicada, VendedorID: 1,
	})

	// Caso 1: el cambio de estado se devuelve correctamente.
	actualizado, err := DB.ActualizarEstadoVehiculo(guardado.ID, models.EstadoVendido)
	if err != nil || actualizado.Estado != models.EstadoVendido {
		t.Fatal("debería devolver el vehículo con el nuevo estado")
	}

	// Caso 2: el cambio queda GUARDADO en el mapa, no solo en la copia.
	leido, _ := DB.BuscarVehiculoPorID(guardado.ID)
	if leido.Estado != models.EstadoVendido {
		t.Fatal("el nuevo estado debe quedar guardado en el repositorio")
	}

	// Caso 3: un vehículo inexistente devuelve ErrNoEncontrado.
	_, err = DB.ActualizarEstadoVehiculo(99, models.EstadoVendido)
	if !errors.Is(err, ErrNoEncontrado) {
		t.Fatal("debería devolver ErrNoEncontrado para un vehículo inexistente")
	}

}

func TestEliminarVehiculo(t *testing.T) {
	InicializarBD()
	vehiculo, _ := DB.GuardarVehiculo(models.Vehiculo{
		Marca: "Renault", Modelo: "Koleos 2018", Precio: 78000000,
		Estado: models.EstadoPublicada, VendedorID: 1,
	})

	// Caso 1: eliminar un vehículo existente.
	if err := DB.EliminarVehiculo(vehiculo.ID); err != nil {
		t.Fatal("debería eliminar el vehículo:", err)
	}
	if _, err := DB.BuscarVehiculoPorID(vehiculo.ID); !errors.Is(err, ErrNoEncontrado) {
		t.Fatal("el vehículo eliminado no debería encontrarse")
	}

	// Caso 2: eliminar un vehículo inexistente.
	if err := DB.EliminarVehiculo(99); !errors.Is(err, ErrNoEncontrado) {
		t.Fatal("debería devolver ErrNoEncontrado para un vehículo inexistente")
	}
}

func TestEliminarUsuario(t *testing.T) {
	InicializarBD()

	// Preparación: Ana con dos vehículos y Luis con uno.
	ana, _ := DB.GuardarUsuario(models.Usuario{NombreUsuario: "ana", Rol: models.RolVendedor})
	luis, _ := DB.GuardarUsuario(models.Usuario{NombreUsuario: "luis", Rol: models.RolVendedor})
	deAna1, _ := DB.GuardarVehiculo(models.Vehiculo{Marca: "Renault", VendedorID: ana.ID})
	deAna2, _ := DB.GuardarVehiculo(models.Vehiculo{Marca: "Mazda", VendedorID: ana.ID})
	deLuis, _ := DB.GuardarVehiculo(models.Vehiculo{Marca: "Kia", VendedorID: luis.ID})

	// Caso 1: eliminar a Ana.
	if err := DB.EliminarUsuario(ana.ID); err != nil {
		t.Fatal("debería eliminar al usuario:", err)
	}
	if _, err := DB.BuscarUsuarioPorID(ana.ID); !errors.Is(err, ErrNoEncontrado) {
		t.Fatal("el usuario eliminado no debería encontrarse")
	}

	// Caso 2: los vehículos de Ana se eliminaron con ella.
	for _, id := range []int{deAna1.ID, deAna2.ID} {
		if _, err := DB.BuscarVehiculoPorID(id); !errors.Is(err, ErrNoEncontrado) {
			t.Fatalf("el vehículo %d de Ana debería haberse eliminado", id)
		}
	}

	// Caso 3: los datos de Luis no se tocaron.
	if _, err := DB.BuscarUsuarioPorID(luis.ID); err != nil {
		t.Fatal("Luis no debería haberse eliminado")
	}
	if _, err := DB.BuscarVehiculoPorID(deLuis.ID); err != nil {
		t.Fatal("el vehículo de Luis no debería haberse eliminado")
	}

	// Caso 4: eliminar un usuario inexistente.
	if err := DB.EliminarUsuario(99); !errors.Is(err, ErrNoEncontrado) {
		t.Fatal("debería devolver ErrNoEncontrado para un usuario inexistente")
	}

}

func TestListarVehiculosPorVendedor(t *testing.T) {
	InicializarBD()
	DB.GuardarVehiculo(models.Vehiculo{Marca: "Renault", VendedorID: 1}) // ID 1
	DB.GuardarVehiculo(models.Vehiculo{Marca: "Kia", VendedorID: 2})     // ID 2
	DB.GuardarVehiculo(models.Vehiculo{Marca: "Mazda", VendedorID: 1})   // ID 3

	// Caso 1: solo los del vendedor 1, ordenados por ID.
	lista := DB.ListarVehiculosPorVendedor(1)
	if len(lista) != 2 || lista[0].ID != 1 || lista[1].ID != 3 {
		t.Fatal("debería devolver solo los vehículos del vendedor 1, ordenados por ID")
	}

	// Caso 2: un vendedor sin vehículos recibe una lista vacía.
	if lista := DB.ListarVehiculosPorVendedor(99); len(lista) != 0 {
		t.Fatal("un vendedor sin vehículos debería recibir una lista vacía")
	}
}

func TestListarUsuarios(t *testing.T) {
	InicializarBD()
	DB.GuardarUsuario(models.Usuario{NombreUsuario: "admin", Rol: models.RolAdministrador})
	DB.GuardarUsuario(models.Usuario{NombreUsuario: "ana", Rol: models.RolVendedor})

	lista := DB.ListarUsuarios()
	if len(lista) != 2 || lista[0].NombreUsuario != "admin" || lista[1].NombreUsuario != "ana" {
		t.Fatal("debería devolver todos los usuarios, ordenados por ID")
	}
}
