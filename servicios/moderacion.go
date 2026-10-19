package servicios

import (
	"automarket/imagenes"
	"automarket/models"
	"automarket/repository"
)

// ListarPendientes devuelve las publicaciones que esperan aprobación,
// ordenadas por ID. La usa la API del administrador.
func ListarPendientes() []models.Vehiculo {
	return repository.DB.ListarVehiculosPorEstado(models.EstadoPendiente)
}

// AutorizarPublicacion cambia una publicación de pendiente a publicada,
// lo que la hace visible en el catálogo público (HU-ADM-01).
//
// Errores: ErrNoEncontrado si no existe; ErrEstadoNoValido si no está pendiente.
func AutorizarPublicacion(id int) (models.Vehiculo, error) {
	vehiculo, err := repository.DB.BuscarVehiculoPorID(id)
	if err != nil {
		return models.Vehiculo{}, err
	}
	if vehiculo.Estado != models.EstadoPendiente {
		return models.Vehiculo{}, ErrEstadoNoValido
	}
	return repository.DB.ActualizarEstadoVehiculo(id, models.EstadoPublicada)
}

// EliminarPublicacion elimina una publicación, sin importar su estado,
// junto con sus fotos.
//
// Errores: ErrNoEncontrado si no existe.
func EliminarPublicacion(id int) error {
	// Se busca primero para saber qué fotos borrar.
	vehiculo, err := repository.DB.BuscarVehiculoPorID(id)
	if err != nil {
		return err
	}

	// Primero los datos, después los archivos. Si borrar los datos falla,
	// las fotos siguen ahí y la publicación sigue completa. Al revés, la
	// publicación quedaría apuntando a fotos que ya no existen.
	if err := repository.DB.EliminarVehiculo(id); err != nil {
		return err
	}
	imagenes.Eliminar(vehiculo.Imagenes)
	return nil
}

// EliminarUsuario elimina a un vendedor junto con todas sus publicaciones
// y sus fotos.
//
// Los administradores no se pueden eliminar: si el único administrador
// se eliminara, nadie podría volver a moderar la plataforma.
//
// Errores: ErrNoEncontrado si no existe; ErrAdministradorProtegido si es administrador.
func EliminarUsuario(id int) error {
	usuario, err := repository.DB.BuscarUsuarioPorID(id)
	if err != nil {
		return err
	}
	if usuario.Rol == models.RolAdministrador {
		return ErrAdministradorProtegido
	}

	// Se anotan las fotos de todas sus publicaciones ANTES de eliminarlo.
	var fotos []string
	for _, vehiculo := range repository.DB.ListarVehiculosPorVendedor(id) {
		fotos = append(fotos, vehiculo.Imagenes...)
	}

	if err := repository.DB.EliminarUsuario(id); err != nil {
		return err
	}
	imagenes.Eliminar(fotos)
	return nil
}

// PanelModeracion reúne todo lo que muestra la página del administrador.
type PanelModeracion struct {
	Pendientes []VehiculoPublico // Esperan autorización
	Publicados []VehiculoPublico // Están en el catálogo
	Vendedores []models.Usuario  // Vendedores registrados
}

// ObtenerPanelModeracion arma el panel de la página de moderación (HU-ADM-01).
func ObtenerPanelModeracion() (PanelModeracion, error) {
	pendientes, err := armarListaPublica(repository.DB.ListarVehiculosPorEstado(models.EstadoPendiente))
	if err != nil {
		return PanelModeracion{}, err
	}

	publicados, err := armarListaPublica(repository.DB.ListarVehiculosPorEstado(models.EstadoPublicada))
	if err != nil {
		return PanelModeracion{}, err
	}

	return PanelModeracion{
		Pendientes: pendientes,
		Publicados: publicados,
		Vendedores: listarVendedores(),
	}, nil
}

// listarVendedores devuelve solo los usuarios con rol vendedor.
// Los administradores no aparecen porque no se pueden eliminar.
func listarVendedores() []models.Usuario {
	vendedores := make([]models.Usuario, 0)
	for _, usuario := range repository.DB.ListarUsuarios() {
		if usuario.Rol == models.RolVendedor {
			vendedores = append(vendedores, usuario)
		}
	}
	return vendedores
}
