/*
	Paquete auth: contiene las herramientas de seguridad para manejar contraseñas.

Regla principal: NUNCA guardamos la contraseña tal como la escribe el usuario.
En su lugar guardamos un "hash", que es como una huella digital de la
contraseña: sirve para comprobarla, pero no se puede revertir para descubrir la contraseña original.
*/
package auth

import (
	"crypto/pbkdf2" // Algoritmo que convierte la contraseña en un hash lento de calcular
	"crypto/rand"   // Generador de números aleatorios seguro (criptografía)
	"crypto/sha256" // Función hash que PBKDF2 usa por dentro
	"crypto/subtle" // Permite comparar bytes siempre en el mismo tiempo
	"encoding/hex"  // Convierte bytes a texto (hexadecimal) y viceversa
	"errors"        // Permite crear nuestros propios errores
	"strings"       // Permite separar el texto guardado en sus dos partes
)

// Valores de configuración del hash.
// Están en minúscula, así que son privados: ningún otro paquete puede cambiarlos.
const (
	// iteraciones: cuántas veces se repite el cálculo del hash.
	// Más repeticiones = más lento para un atacante que intente adivinar.
	// 600.000 es el valor que recomienda OWASP para PBKDF2 con SHA-256.
	iteraciones = 600_000

	// largoSal: tamaño en bytes de la sal (16 bytes = 128 bits).
	largoSal = 16

	// largoHash: tamaño en bytes del hash resultante (32 bytes = 256 bits).
	largoHash = 32

	// separador: carácter que separa la sal del hash en el texto guardado.
	// Ejemplo de texto guardado: "a1b2c3...$d4e5f6..."
	separador = "$"
)

// ErrHashInvalido se devuelve cuando el texto guardado no tiene el formato
// esperado "sal$hash". Esto indica un dato dañado, no una contraseña incorrecta.
var ErrHashInvalido = errors.New("formato de hash inválido")

// GenerarHash recibe la contraseña del usuario y devuelve el texto que se
// debe guardar en la base de datos, con el formato "salHex$hashHex".
//
// Se usa al REGISTRAR un usuario.
func GenerarHash(contrasena string) (string, error) {
	// Paso 1: crear la sal.
	// La sal son bytes aleatorios distintos para cada usuario. Gracias a ella,
	// dos personas con la misma contraseña tendrán hashes diferentes.
	sal := make([]byte, largoSal) // Reserva espacio para 16 bytes
	rand.Read(sal)                // Llena ese espacio con bytes aleatorios

	// Paso 2: calcular el hash.
	// PBKDF2 mezcla la contraseña con la sal y repite el proceso
	// "iteraciones" veces usando SHA-256.
	hash, err := pbkdf2.Key(sha256.New, contrasena, sal, iteraciones, largoHash)
	if err != nil {
		// Si algo falla, devolvemos texto vacío y el error para que
		// quien llamó a esta función decida qué hacer.
		return "", err
	}

	// Paso 3: unir sal y hash en un solo texto.
	// Los bytes no se pueden guardar cómodamente como texto, así que los
	// convertimos a hexadecimal (solo caracteres 0-9 y a-f).
	// Guardamos la sal junto al hash porque la necesitaremos para verificar.
	return hex.EncodeToString(sal) + separador + hex.EncodeToString(hash), nil
}

// VerificarContrasena comprueba si la contraseña que escribió el usuario
// corresponde al hash guardado.
//
// Se usa al AUTENTICAR un usuario (en cada petición, porque usamos Basic Auth).
//
// Posibles resultados:
//   - (true, nil):  la contraseña es correcta.
//   - (false, nil): la contraseña es incorrecta (situación normal).
//   - (false, error): el hash guardado está dañado (problema del sistema).
func VerificarContrasena(contrasena, guardado string) (bool, error) {
	// Paso 1: separar el texto guardado en sal y hash.
	// strings.Cut corta el texto en el primer "$" y nos dice si lo encontró.
	salHex, hashHex, encontrado := strings.Cut(guardado, separador)
	if !encontrado {
		return false, ErrHashInvalido
	}

	// Paso 2: convertir la sal de texto hexadecimal a bytes.
	sal, err := hex.DecodeString(salHex)
	if err != nil {
		return false, ErrHashInvalido
	}

	// Paso 3: convertir el hash guardado de texto hexadecimal a bytes.
	esperado, err := hex.DecodeString(hashHex)
	if err != nil {
		return false, ErrHashInvalido
	}

	// Paso 4: recalcular el hash con la contraseña recibida y la MISMA sal.
	// Si la contraseña es correcta, el resultado será idéntico al guardado.
	calculado, err := pbkdf2.Key(sha256.New, contrasena, sal, iteraciones, len(esperado))
	if err != nil {
		return false, err
	}

	// Paso 5: comparar ambos hashes en tiempo constante.
	// Una comparación normal se detiene en el primer byte diferente, y un
	// atacante podría medir esos tiempos para obtener pistas.
	// ConstantTimeCompare siempre tarda lo mismo y devuelve 1 si son iguales.
	return subtle.ConstantTimeCompare(calculado, esperado) == 1, nil
}
