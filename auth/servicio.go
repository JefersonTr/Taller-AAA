package auth

import (
	"errors"

	"automarket/models"
	"automarket/repository"
)

// ErrCredencialesInvalidas se devuelve cuando el usuario no existe o la
// contraseña es incorrecta. Usamos el MISMO error para ambos casos a
// propósito: si la respuesta dijera "el usuario no existe", un atacante
// podría descubrir qué nombres de usuario están registrados.
var ErrCredencialesInvalidas = errors.New("credenciales inválidas")

// RepositorioUsuarios describe lo único que el servicio necesita del
// repositorio: buscar un usuario por su nombre de usuario.
//
// Cualquier tipo que tenga este método cumple la interfaz automáticamente.
// El repositorio real (repository.BaseDatosMemoria) ya la cumple, y en las
// pruebas usamos uno falso.
type RepositorioUsuarios interface {
	BuscarUsuarioPorNombreUsuario(nombreUsuario string) (models.Usuario, error)
}

// Servicio contiene la lógica de autenticación.
type Servicio struct {
	usuarios RepositorioUsuarios
}

// NuevoServicio crea el servicio de autenticación con el repositorio indicado.
// Recibir el repositorio como parámetro se llama "inyección de dependencias".
func NuevoServicio(usuarios RepositorioUsuarios) *Servicio {
	return &Servicio{usuarios: usuarios}
}

// Autenticar verifica el nombre de usuario y la contraseña.
//
// Posibles resultados:
//   - (usuario, nil): las credenciales son correctas.
//   - (vacío, ErrCredencialesInvalidas): usuario inexistente o contraseña incorrecta.
//   - (vacío, otro error): falla del sistema (por ejemplo, hash guardado dañado).
func (s *Servicio) Autenticar(nombreUsuario, contrasena string) (models.Usuario, error) {
	// Paso 1: buscar al usuario.
	usuario, err := s.usuarios.BuscarUsuarioPorNombreUsuario(nombreUsuario)
	if errors.Is(err, repository.ErrNoEncontrado) {
		return models.Usuario{}, ErrCredencialesInvalidas
	}
	if err != nil {
		return models.Usuario{}, err
	}

	// Paso 2: comprobar la contraseña contra el hash guardado.
	valida, err := VerificarContrasena(contrasena, usuario.HashContrasena)
	if err != nil {
		return models.Usuario{}, err
	}
	if !valida {
		return models.Usuario{}, ErrCredencialesInvalidas
	}

	// Paso 3: credenciales correctas, devolvemos el usuario.
	return usuario, nil
}
