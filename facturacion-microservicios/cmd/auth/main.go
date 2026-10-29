// Servicio de Autenticación - Puerto 8081
// Gestiona registro con rol (admin/cliente), login y consulta de usuarios, sin JWT.
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

	log.Printf("🔐 Servicio de Autenticación iniciando en puerto %s", port)
	log.Printf("   Archivo de datos: %s", dataFile)

	// Crear store de usuarios (IDs nuevos desde 3)
	store, err := jsondb.New[auth.Usuario](dataFile, 3,
		func(u *auth.Usuario) int { return u.ID },
		func(u *auth.Usuario, id int) { u.ID = id })
	if err != nil {
		log.Fatalf("Error inicializando store de usuarios: %v", err)
	}

	handler := auth.NewHandler(store)
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

	log.Printf("🔐 Servicio de Autenticación escuchando en http://localhost:%s", port)
	if err := server.ListenAndServe(); err != http.ErrServerClosed {
		log.Fatalf("Error en servidor: %v", err)
	}
	log.Println("Servicio de autenticación detenido correctamente")
}

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}
