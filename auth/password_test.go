/*
	Pruebas automáticas del paquete auth.

Go reconoce como pruebas los archivos que terminan en "_test.go"
y las funciones que empiezan con "Test". Se ejecutan con: go test ./auth/ -v
*/
package auth

import "testing" // Paquete estándar de Go para escribir pruebas

// TestGenerarHashYVerificar comprueba que el hash de contraseñas
// funcione correctamente en los casos más importantes.
//
// "t" es el objeto que usa Go para reportar si la prueba pasa o falla.
// t.Fatal detiene la prueba inmediatamente y la marca como fallida.
func TestGenerarHashYVerificar(t *testing.T) {
	// Caso 1: generar un hash no debe producir error.
	hash, err := GenerarHash("secreta123")
	if err != nil {
		t.Fatal("no se pudo generar el hash:", err)
	}

	// Caso 2: la contraseña correcta debe ser aceptada.
	correcta, err := VerificarContrasena("secreta123", hash)
	if err != nil || !correcta {
		t.Fatal("debería aceptar la contraseña correcta")
	}

	// Caso 3: una contraseña incorrecta debe ser rechazada.
	// El guion bajo "_" significa que ignoramos el error en este caso.
	incorrecta, _ := VerificarContrasena("otra", hash)
	if incorrecta {
		t.Fatal("debería rechazar una contraseña incorrecta")
	}

	// Caso 4: la misma contraseña debe generar hashes distintos.
	// Esto demuestra que la sal aleatoria está funcionando.
	otroHash, _ := GenerarHash("secreta123")
	if hash == otroHash {
		t.Fatal("la misma contraseña debe producir hashes distintos (por la sal)")
	}

	// Caso 5: un texto guardado sin el formato "sal$hash" debe dar error.
	_, err = VerificarContrasena("secreta123", "texto-sin-formato")
	if err != ErrHashInvalido {
		t.Fatal("debería devolver ErrHashInvalido con un formato incorrecto")
	}
}
