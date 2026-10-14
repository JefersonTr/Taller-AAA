package auth

import (
	"errors"
	"testing"

	"automarket/models"
	"automarket/repository"
)

// repositorioFalso simula el repositorio usando un mapa sencillo.
// Cumple la interfaz RepositorioUsuarios porque tiene el método requerido.
type repositorioFalso struct {
	usuarios map[string]models.Usuario
}

func (r repositorioFalso) BuscarUsuarioPorNombreUsuario(nombreUsuario string) (models.Usuario, error) {
	usuario, existe := r.usuarios[nombreUsuario]
	if !existe {
		return models.Usuario{}, repository.ErrNoEncontrado
	}
	return usuario, nil
}

// TestAutenticar comprueba los tres casos posibles de la autenticación.
func TestAutenticar(t *testing.T) {
	// Preparación: un usuario registrado con contraseña "secreta123".
	hash, err := GenerarHash("secreta123")
	if err != nil {
		t.Fatal(err)
	}
	repo := repositorioFalso{usuarios: map[string]models.Usuario{
		"ana": {ID: 1, NombreUsuario: "ana", Rol: models.RolVendedor, HashContrasena: hash},
	}}
	servicio := NuevoServicio(repo)

	// Caso 1: credenciales correctas devuelven el usuario.
	usuario, err := servicio.Autenticar("ana", "secreta123")
	if err != nil || usuario.ID != 1 {
		t.Fatal("debería autenticar con credenciales correctas")
	}

	// Caso 2: contraseña incorrecta.
	_, err = servicio.Autenticar("ana", "otra")
	if !errors.Is(err, ErrCredencialesInvalidas) {
		t.Fatal("debería rechazar una contraseña incorrecta")
	}

	// Caso 3: usuario inexistente devuelve el MISMO error que el caso 2.
	_, err = servicio.Autenticar("pedro", "secreta123")
	if !errors.Is(err, ErrCredencialesInvalidas) {
		t.Fatal("debería rechazar un usuario inexistente con el mismo error")
	}
}
