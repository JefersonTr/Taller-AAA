package servicios

import (
	"automarket/models"
	"automarket/repository"
)

// ListarPendientes devuelve las publicaciones que esperan aprobación,
// ordenadas por ID. Las necesita la pantalla de moderación del administrador.
func ListarPendientes() []models.Vehiculo {
	return repository.DB.ListarVehiculosPorEstado(models.EstadoPendiente)
}

// AutorizarPublicacion cambia una publicación de pendiente a publicada,
// lo que la hace visible en el catálogo público (HU-ADM-01).
//
// Errores: ErrNoEncontrado si no existe; ErrEstadoNoValido si no está pendiente.
func AutorizarPublicacion(id int) (models.Vehiculo, error) {
	// Paso 1: buscar la publicación.
	vehiculo, err := repository.DB.BuscarVehiculoPorID(id)
	if err != nil {
		return models.Vehiculo{}, err
	}

	// Paso 2: regla de negocio, solo se autoriza lo que está pendiente.
	if vehiculo.Estado != models.EstadoPendiente {
		return models.Vehiculo{}, ErrEstadoNoValido
	}

	// Paso 3: cambiar el estado con la función que creamos en la Parte 3.
	return repository.DB.ActualizarEstadoVehiculo(id, models.EstadoPublicada)
}

// EliminarPublicacion elimina una publicación, sin importar su estado.
//
// Errores: ErrNoEncontrado si no existe.
func EliminarPublicacion(id int) error {
	return repository.DB.EliminarVehiculo(id)
}

// EliminarUsuario elimina a un vendedor junto con todas sus publicaciones.
//
// Los administradores no se pueden eliminar: si el único administrador
// se eliminara, nadie podría volver a moderar la plataforma.
//
// Errores: ErrNoEncontrado si no existe; ErrAdministradorProtegido si es administrador.
func EliminarUsuario(id int) error {
	// Paso 1: buscar al usuario para conocer su rol.
	usuario, err := repository.DB.BuscarUsuarioPorID(id)
	if err != nil {
		return err
	}

	// Paso 2: regla de negocio, los administradores están protegidos.
	if usuario.Rol == models.RolAdministrador {
		return ErrAdministradorProtegido
	}

	// Paso 3: eliminar al usuario y sus vehículos (lo hace el repositorio).
	return repository.DB.EliminarUsuario(id)
}
