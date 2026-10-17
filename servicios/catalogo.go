package servicios

import (
	"errors"

	"automarket/models"
	"automarket/repository"
)

// ContactoVendedor contiene los únicos datos del vendedor que puede ver
// un visitante. No incluye nombre de usuario, rol ni contraseña.
type ContactoVendedor struct {
	Nombre   string `json:"nombre"`
	Telefono string `json:"telefono"`
}

// VehiculoPublico es la información de un vehículo que ve un visitante.
// No incluye el estado ni el ID interno del vendedor.
type VehiculoPublico struct {
	ID       int              `json:"id"`
	Marca    string           `json:"marca"`
	Modelo   string           `json:"modelo"`
	Precio   int              `json:"precio"`
	Vendedor ContactoVendedor `json:"vendedor"`
}

// ListarCatalogo devuelve los vehículos publicados con el contacto de su
// vendedor (HU-VIS-01).
func ListarCatalogo() ([]VehiculoPublico, error) {
	return armarListaPublica(repository.DB.ListarVehiculosPorEstado(models.EstadoPublicada))
}

// ConsultarVehiculo devuelve un vehículo publicado con el contacto de su
// vendedor (HU-VIS-01).
//
// Un vehículo que no está publicado devuelve ErrNoEncontrado, igual que uno
// inexistente: el visitante no debe saber que hay publicaciones sin aprobar.
func ConsultarVehiculo(id int) (VehiculoPublico, error) {
	vehiculo, err := repository.DB.BuscarVehiculoPorID(id)
	if err != nil {
		return VehiculoPublico{}, err
	}
	if vehiculo.Estado != models.EstadoPublicada {
		return VehiculoPublico{}, ErrNoEncontrado
	}
	return armarVehiculoPublico(vehiculo)
}

// armarListaPublica agrega el contacto del vendedor a cada vehículo de la
// lista. Los vehículos cuyo vendedor ya no existe se omiten.
func armarListaPublica(vehiculos []models.Vehiculo) ([]VehiculoPublico, error) {
	lista := make([]VehiculoPublico, 0, len(vehiculos))
	for _, vehiculo := range vehiculos {
		publico, err := armarVehiculoPublico(vehiculo)
		if errors.Is(err, ErrNoEncontrado) {
			// El vendedor ya no existe: el vehículo no se muestra.
			continue
		}
		if err != nil {
			return nil, err
		}
		lista = append(lista, publico)
	}
	return lista, nil
}

// armarVehiculoPublico combina un vehículo con el contacto de su vendedor.
func armarVehiculoPublico(vehiculo models.Vehiculo) (VehiculoPublico, error) {
	vendedor, err := repository.DB.BuscarUsuarioPorID(vehiculo.VendedorID)
	if err != nil {
		return VehiculoPublico{}, err
	}

	return VehiculoPublico{
		ID:     vehiculo.ID,
		Marca:  vehiculo.Marca,
		Modelo: vehiculo.Modelo,
		Precio: vehiculo.Precio,
		Vendedor: ContactoVendedor{
			Nombre:   vendedor.Nombre,
			Telefono: vendedor.Telefono,
		},
	}, nil
}
