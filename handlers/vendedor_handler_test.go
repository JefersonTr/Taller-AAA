package handlers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"automarket/auth"
	"automarket/middleware"
	"automarket/models"
	"automarket/repository"

	"github.com/gin-gonic/gin"
)

// autenticadorPrueba acepta a los usuarios del mapa con la contraseña "clave".
// Cumple la interfaz middleware.Autenticador.
type autenticadorPrueba map[string]models.Usuario

func (a autenticadorPrueba) Autenticar(nombreUsuario, contrasena string) (models.Usuario, error) {
	usuario, existe := a[nombreUsuario]
	if !existe || contrasena != "clave" {
		return models.Usuario{}, auth.ErrCredencialesInvalidas
	}
	return usuario, nil
}

func TestReportarVenta(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repository.InicializarBD()

	// Dos vendedores: Ana es dueña de los vehículos, Luis no.
	ana := models.Usuario{ID: 1, NombreUsuario: "ana", Rol: models.RolVendedor}
	luis := models.Usuario{ID: 2, NombreUsuario: "luis", Rol: models.RolVendedor}

	// Vehículo 1: publicado. Vehículo 2: pendiente. Ambos de Ana.
	repository.DB.GuardarVehiculo(models.Vehiculo{
		Marca: "Renault", Modelo: "Koleos 2018", Precio: 78000000,
		Estado: models.EstadoPublicada, VendedorID: ana.ID,
	})
	repository.DB.GuardarVehiculo(models.Vehiculo{
		Marca: "Mazda", Modelo: "3 2020", Precio: 65000000,
		Estado: models.EstadoPendiente, VendedorID: ana.ID,
	})

	// Misma cadena de middlewares que en main.go.
	r := gin.New()
	r.PUT("/vehiculos/:id/venta",
		middleware.RequiereAutenticacion(autenticadorPrueba{"ana": ana, "luis": luis}),
		middleware.RequiereRol(models.RolVendedor),
		ReportarVenta,
	)

	// El ORDEN de los casos importa: "venta exitosa" cambia el estado del
	// vehículo 1, y el último caso comprueba que no se pueda vender dos veces.
	casos := []struct {
		nombre         string
		usuario        string
		ruta           string
		codigoEsperado int
	}{
		{"vendedor que no es dueño", "luis", "/vehiculos/1/venta", http.StatusForbidden},
		{"vehículo pendiente", "ana", "/vehiculos/2/venta", http.StatusConflict},
		{"vehículo inexistente", "ana", "/vehiculos/99/venta", http.StatusNotFound},
		{"identificador inválido", "ana", "/vehiculos/abc/venta", http.StatusBadRequest},
		{"venta exitosa", "ana", "/vehiculos/1/venta", http.StatusOK},
		{"vehículo ya vendido", "ana", "/vehiculos/1/venta", http.StatusConflict},
	}

	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			peticion := httptest.NewRequest(http.MethodPut, caso.ruta, nil)
			peticion.SetBasicAuth(caso.usuario, "clave")
			respuesta := httptest.NewRecorder()
			r.ServeHTTP(respuesta, peticion)

			if respuesta.Code != caso.codigoEsperado {
				t.Fatalf("código esperado %d, obtenido %d: %s",
					caso.codigoEsperado, respuesta.Code, respuesta.Body.String())
			}
		})
	}

	// La venta debe quedar guardada en el repositorio.
	vehiculo, _ := repository.DB.BuscarVehiculoPorID(1)
	if vehiculo.Estado != models.EstadoVendido {
		t.Fatal("el vehículo vendido debe quedar en estado vendido")
	}
}

// peticionJSON crea una petición HTTP con un cuerpo JSON.
func peticionJSON(metodo, ruta, cuerpo string) *http.Request {
	peticion := httptest.NewRequest(metodo, ruta, strings.NewReader(cuerpo))
	peticion.Header.Set("Content-Type", "application/json")
	return peticion
}

func TestRegistrarVendedor(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repository.InicializarBD()

	r := gin.New()
	r.POST("/registro", RegistrarVendedor)

	// El ORDEN importa: el caso "nombre de usuario repetido" depende de
	// que el primer caso ya haya registrado a "ana".
	casos := []struct {
		nombre         string
		cuerpo         string
		codigoEsperado int
	}{
		{"registro válido",
			`{"nombre":"Ana Gómez","nombre_usuario":"ana","telefono":"3001234567","contrasena":"secreta123"}`,
			http.StatusCreated},
		{"nombre de usuario repetido",
			`{"nombre":"Otra Ana","nombre_usuario":"ana","telefono":"3000000000","contrasena":"otra123"}`,
			http.StatusConflict},
		{"sin contraseña",
			`{"nombre":"Luis","nombre_usuario":"luis","telefono":"3000000000"}`,
			http.StatusBadRequest},
		{"nombre de usuario con dos puntos",
			`{"nombre":"Luis","nombre_usuario":"luis:p","telefono":"3000000000","contrasena":"secreta123"}`,
			http.StatusBadRequest},
		{"JSON mal formado", `{"nombre":`, http.StatusBadRequest},
	}

	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			respuesta := httptest.NewRecorder()
			r.ServeHTTP(respuesta, peticionJSON(http.MethodPost, "/registro", caso.cuerpo))

			if respuesta.Code != caso.codigoEsperado {
				t.Fatalf("código esperado %d, obtenido %d: %s",
					caso.codigoEsperado, respuesta.Code, respuesta.Body.String())
			}
			// Seguridad: la contraseña nunca debe volver en la respuesta.
			if strings.Contains(respuesta.Body.String(), "secreta123") {
				t.Fatal("la respuesta no debe incluir la contraseña")
			}
		})
	}

	// Seguridad: la contraseña se guarda como hash, no en texto plano.
	ana, err := repository.DB.BuscarUsuarioPorNombreUsuario("ana")
	if err != nil {
		t.Fatal("el vendedor registrado debería existir:", err)
	}
	if ana.HashContrasena == "secreta123" {
		t.Fatal("la contraseña no debe guardarse en texto plano")
	}
	if valida, _ := auth.VerificarContrasena("secreta123", ana.HashContrasena); !valida {
		t.Fatal("el hash guardado debe corresponder a la contraseña registrada")
	}
}

// TestRegistroNoPermiteElegirRol simula un ataque de asignación masiva:
// alguien intenta registrarse como administrador enviando campos extra.
func TestRegistroNoPermiteElegirRol(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repository.InicializarBD()

	r := gin.New()
	r.POST("/registro", RegistrarVendedor)

	cuerpo := `{"nombre":"Mallory","nombre_usuario":"mallory","telefono":"3000000000",
		"contrasena":"secreta123","rol":"administrador","id":99}`
	respuesta := httptest.NewRecorder()
	r.ServeHTTP(respuesta, peticionJSON(http.MethodPost, "/registro", cuerpo))

	if respuesta.Code != http.StatusCreated {
		t.Fatalf("código esperado 201, obtenido %d", respuesta.Code)
	}

	mallory, err := repository.DB.BuscarUsuarioPorNombreUsuario("mallory")
	if err != nil {
		t.Fatal(err)
	}
	if mallory.Rol != models.RolVendedor {
		t.Fatal("un cliente no debe poder elegir su rol al registrarse")
	}
	if mallory.ID == 99 {
		t.Fatal("un cliente no debe poder elegir su ID")
	}
}

func TestPublicarVehiculo(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repository.InicializarBD()

	ana := models.Usuario{ID: 1, NombreUsuario: "ana", Rol: models.RolVendedor}

	// Misma cadena de middlewares que en main.go.
	r := gin.New()
	r.POST("/vehiculos",
		middleware.RequiereAutenticacion(autenticadorPrueba{"ana": ana}),
		middleware.RequiereRol(models.RolVendedor),
		PublicarVehiculo,
	)

	casos := []struct {
		nombre         string
		cuerpo         string
		codigoEsperado int
	}{
		// El primer cuerpo incluye campos que el cliente NO debe controlar:
		// intenta publicarse directamente y a nombre de otro vendedor.
		{"publicación válida con campos extra",
			`{"marca":"Renault","modelo":"Koleos 2018","precio":78000000,"estado":"publicada","vendedor_id":99}`,
			http.StatusCreated},
		{"precio cero", `{"marca":"Renault","modelo":"Koleos 2018","precio":0}`, http.StatusBadRequest},
		{"precio negativo", `{"marca":"Renault","modelo":"Koleos 2018","precio":-5}`, http.StatusBadRequest},
		{"sin marca", `{"modelo":"Koleos 2018","precio":78000000}`, http.StatusBadRequest},
	}

	for _, caso := range casos {
		t.Run(caso.nombre, func(t *testing.T) {
			peticion := peticionJSON(http.MethodPost, "/vehiculos", caso.cuerpo)
			peticion.SetBasicAuth("ana", "clave")
			respuesta := httptest.NewRecorder()
			r.ServeHTTP(respuesta, peticion)

			if respuesta.Code != caso.codigoEsperado {
				t.Fatalf("código esperado %d, obtenido %d: %s",
					caso.codigoEsperado, respuesta.Code, respuesta.Body.String())
			}
		})
	}

	// Criterio de aceptación de HU-VEN-01: la publicación queda pendiente
	// y asociada a la cuenta autenticada, ignorando lo que envió el cliente.
	vehiculo, err := repository.DB.BuscarVehiculoPorID(1)
	if err != nil {
		t.Fatal("la publicación válida debería existir:", err)
	}
	if vehiculo.Estado != models.EstadoPendiente {
		t.Fatal("toda publicación debe iniciar en estado pendiente")
	}
	if vehiculo.VendedorID != ana.ID {
		t.Fatal("la publicación debe asociarse al vendedor autenticado")
	}
}
