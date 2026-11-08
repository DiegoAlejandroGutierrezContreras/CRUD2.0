// Servicio de Productos - Puerto 8082
// Gestiona el CRUD de productos con precios y stock.
package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"facturacion-microservicios/internal/jsondb"
	"facturacion-microservicios/internal/middleware"
	"facturacion-microservicios/internal/productos"
)

func main() {
	port := getEnv("PORT", "8082")
	dataFile := getEnv("DATA_FILE", "data/productos.json")
	authURL := getEnv("AUTH_URL", "http://localhost:8081")

	log.Printf("Servicio de Productos iniciando en puerto %s", port)
	log.Printf("   Archivo de datos: %s", dataFile)
	log.Printf("   Auth URL: %s", authURL)

	// Crear store de productos. El ID inicial es solo una semilla: jsondb sube
	// nextID al máximo ID que ya exista en el archivo.
	store, err := jsondb.New[productos.Producto](dataFile, 104,
		func(p *productos.Producto) int { return p.ID },
		func(p *productos.Producto, id int) { p.ID = id })
	if err != nil {
		log.Fatalf("Error inicializando store de productos: %v", err)
	}

	handler := productos.NewHandler(store, authURL)
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
		log.Println("Apagando servicio de productos...")
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		server.Shutdown(ctx)
	}()

	log.Printf(" Servicio de Productos escuchando en http://localhost:%s", port)
	if err := server.ListenAndServe(); err != http.ErrServerClosed {
		log.Fatalf("Error en servidor: %v", err)
	}
	log.Println("Servicio de productos detenido correctamente")
}

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}
