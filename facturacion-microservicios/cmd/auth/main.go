// Servicio de Autenticación - Puerto 8081
// Gestiona registro con rol (admin/cliente), login con token de sesión opaco,
// logout y consulta de usuarios. No usa JWT.
package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"facturacion-microservicios/internal/auth"
	"facturacion-microservicios/internal/jsondb"
	"facturacion-microservicios/internal/middleware"
)

func main() {
	port := getEnv("PORT", "8081")
	dataFile := getEnv("DATA_FILE", "data/usuarios.json")
	clientesURL := getEnv("CLIENTES_URL", "http://localhost:8083")
	internalToken := getEnv("INTERNAL_TOKEN", "facturacion-interno")

	log.Printf("Servicio de Autenticación iniciando en puerto %s", port)
	log.Printf("   Archivo de datos: %s", dataFile)
	log.Printf("   Clientes URL: %s", clientesURL)

	// Crear store de usuarios. El ID inicial es solo una semilla: jsondb sube
	// nextID al máximo ID que ya exista en el archivo.
	store, err := jsondb.New[auth.Usuario](dataFile, 3,
		func(u *auth.Usuario) int { return u.ID },
		func(u *auth.Usuario, id int) { u.ID = id })
	if err != nil {
		log.Fatalf("Error inicializando store de usuarios: %v", err)
	}

	sessions := auth.NewSessionStore(ttlDesdeEnv("SESSION_TTL", auth.SessionTTLPorDefecto))
	handler := auth.NewHandlerConClientes(store, sessions, auth.NewClientesRegistro(clientesURL, internalToken))
	mux := http.NewServeMux()
	handler.RegisterRoutes(mux)

	// Aplicar solo CORS
	finalHandler := middleware.CORS(mux)

	server := &http.Server{
		Addr:         ":" + port,
		Handler:      finalHandler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Apagado ordenado
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		<-sigChan
		log.Println("Apagando servicio de autenticación...")
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		server.Shutdown(ctx)
	}()

	log.Printf("Servicio de Autenticación escuchando en http://localhost:%s", port)
	if err := server.ListenAndServe(); err != http.ErrServerClosed {
		log.Fatalf("Error en servidor: %v", err)
	}
	log.Println("Servicio de autenticación detenido correctamente")
}

// ttlDesdeEnv lee una duración de la variable de entorno y usa el valor por
// defecto si no existe o no es un duration válido (por ejemplo "8h", "30m").
func ttlDesdeEnv(key string, defaultVal time.Duration) time.Duration {
	valor := os.Getenv(key)
	if valor == "" {
		return defaultVal
	}
	ttl, err := time.ParseDuration(valor)
	if err != nil || ttl <= 0 {
		log.Printf("SESSION_TTL inválido (%q); se usa %s", valor, defaultVal)
		return defaultVal
	}
	return ttl
}

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}
