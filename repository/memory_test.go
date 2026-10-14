package repository

import (
	"errors"
	"testing"

	"automarket/models"
)

func TestActualizarEstadoVehiculo(t *testing.T) {
	// Preparación: base de datos limpia con un vehículo publicado.
	InicializarBD()
	guardado := DB.GuardarVehiculo(models.Vehiculo{
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
