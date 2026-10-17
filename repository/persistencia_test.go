package repository

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"automarket/models"
)

func TestPersistenciaDatos(t *testing.T) {
	archivo := filepath.Join(t.TempDir(), "datos.json")

	// Primera "ejecución": una usuaria y dos vehículos; el segundo se elimina.
	InicializarBD()
	if err := DB.AbrirArchivoDatos(archivo); err != nil {
		t.Fatal(err)
	}
	ana, err := DB.GuardarUsuario(models.Usuario{
		NombreUsuario: "ana", Rol: models.RolVendedor, HashContrasena: "sal$hash",
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := DB.GuardarVehiculo(models.Vehiculo{Marca: "Renault", VendedorID: ana.ID}); err != nil {
		t.Fatal(err)
	}
	temporal, _ := DB.GuardarVehiculo(models.Vehiculo{Marca: "Kia", VendedorID: ana.ID})
	if err := DB.EliminarVehiculo(temporal.ID); err != nil {
		t.Fatal(err)
	}

	// Segunda "ejecución": todo se recupera desde el archivo.
	InicializarBD()
	if err := DB.AbrirArchivoDatos(archivo); err != nil {
		t.Fatal("un archivo válido debe cargarse:", err)
	}

	// Caso 1: la usuaria existe y conserva su hash. Sin el hash, no podría
	// iniciar sesión después de reiniciar (la trampa del json:"-").
	recuperada, err := DB.BuscarUsuarioPorNombreUsuario("ana")
	if err != nil {
		t.Fatal("la usuaria debe sobrevivir al reinicio")
	}
	if recuperada.HashContrasena != "sal$hash" {
		t.Fatal("el hash de la contraseña debe guardarse en el archivo")
	}

	// Caso 2: el vehículo 1 existe; el 2, que se eliminó, no.
	if _, err := DB.BuscarVehiculoPorID(1); err != nil {
		t.Fatal("el vehículo guardado debe sobrevivir al reinicio")
	}
	if _, err := DB.BuscarVehiculoPorID(2); !errors.Is(err, ErrNoEncontrado) {
		t.Fatal("el vehículo eliminado no debe reaparecer")
	}

	// Caso 3: los IDs no se reutilizan; el siguiente vehículo es el 3, no el 2.
	nuevo, _ := DB.GuardarVehiculo(models.Vehiculo{Marca: "Mazda", VendedorID: ana.ID})
	if nuevo.ID != 3 {
		t.Fatalf("el siguiente vehículo debería tener ID 3, tiene %d", nuevo.ID)
	}
}

func TestRestaurarSiFallaElGuardado(t *testing.T) {
	directorio := filepath.Join(t.TempDir(), "datos")

	InicializarBD()
	if err := DB.AbrirArchivoDatos(filepath.Join(directorio, "datos.json")); err != nil {
		t.Fatal(err)
	}

	// Simulamos que el disco falla: borramos la carpeta de datos.
	if err := os.RemoveAll(directorio); err != nil {
		t.Fatal(err)
	}

	// Caso 1: guardar falla y el usuario NO queda en memoria.
	if _, err := DB.GuardarUsuario(models.Usuario{NombreUsuario: "ana"}); err == nil {
		t.Fatal("guardar debería fallar si no se puede escribir en el disco")
	}
	if _, err := DB.BuscarUsuarioPorNombreUsuario("ana"); !errors.Is(err, ErrNoEncontrado) {
		t.Fatal("lo que no se guardó en disco no debe quedar en memoria")
	}

	// Caso 2: el disco se recupera y el ID 1 sigue disponible, porque el
	// contador también se restauró.
	if err := os.MkdirAll(directorio, 0o700); err != nil {
		t.Fatal(err)
	}
	usuario, err := DB.GuardarUsuario(models.Usuario{NombreUsuario: "ana"})
	if err != nil || usuario.ID != 1 {
		t.Fatal("después de restaurar, el primer usuario debe tener ID 1")
	}
}
