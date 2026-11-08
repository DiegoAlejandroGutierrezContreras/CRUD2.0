// Servicio de Clientes - Puerto 8083
// Gestiona el CRUD de clientes con nombre, cédula, correo y dirección.
package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"facturacion-microservicios/internal/clientes"
	"facturacion-microservicios/internal/jsondb"
	"facturacion-microservicios/internal/middleware"
)

func main() {
	port := getEnv("PORT", "8083")
	dataFile := getEnv("DATA_FILE", "data/clientes.json")
	authURL := getEnv("AUTH_URL", "http://localhost:8081")
	internalToken := getEnv("INTERNAL_TOKEN", "facturacion-interno")

	log.Printf("Servicio de Clientes iniciando en puerto %s", port)
	log.Printf("   Archivo de datos: %s", dataFile)
	log.Printf("   Auth URL: %s", authURL)

	// Crear store de clientes. El ID inicial es solo una semilla: jsondb sube
	// nextID al máximo ID que ya exista en el archivo.
	store, err := jsondb.New[clientes.Cliente](dataFile, 203,
		func(c *clientes.Cliente) int { return c.ID },
		func(c *clientes.Cliente, id int) { c.ID = id })
	if err != nil {
		log.Fatalf("Error inicializando store de clientes: %v", err)
	}

	handler := clientes.NewHandlerConInterno(store, authURL, internalToken)
	mux := http.NewServeMux()
	handler.RegisterRoutes(mux)

	// Lectura con sesión; mutaciones solo para administradores.
	finalHandler := middleware.CORS(mux)

	server := &http.Server{
		Addr:         ":" + port,
		Handler:      finalHandler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		<-sigChan
		log.Println("Apagando servicio de clientes...")
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		server.Shutdown(ctx)
	}()

	log.Printf("Servicio de Clientes escuchando en http://localhost:%s", port)
	if err := server.ListenAndServe(); err != http.ErrServerClosed {
		log.Fatalf("Error en servidor: %v", err)
	}
	log.Println("Servicio de clientes detenido correctamente")
}

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}
