// Servicio de Facturación - Puerto 8084
// Consume los servicios de Autenticación, Productos y Clientes para generar facturas.
package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"facturacion-microservicios/internal/facturacion"
	"facturacion-microservicios/internal/jsondb"
	"facturacion-microservicios/internal/middleware"
)

func main() {
	port := getEnv("PORT", "8084")
	dataFile := getEnv("DATA_FILE", "data/facturas.json")
	authURL := getEnv("AUTH_URL", "http://localhost:8081")
	productosURL := getEnv("PRODUCTOS_URL", "http://localhost:8082")
	clientesURL := getEnv("CLIENTES_URL", "http://localhost:8083")

	log.Printf("Servicio de Facturación iniciando en puerto %s", port)
	log.Printf("   Archivo de datos: %s", dataFile)
	log.Printf("   Auth URL: %s", authURL)
	log.Printf("   Productos URL: %s", productosURL)
	log.Printf("   Clientes URL: %s", clientesURL)

	// Crear store de facturas. El ID inicial es solo una semilla: jsondb sube
	// nextID al máximo factura_id que ya exista en el archivo.
	store, err := jsondb.New[facturacion.Factura](dataFile, 5003,
		func(f *facturacion.Factura) int { return f.FacturaID },
		func(f *facturacion.Factura, id int) { f.FacturaID = id })
	if err != nil {
		log.Fatalf("Error inicializando store de facturas: %v", err)
	}

	urls := facturacion.ServiceURLs{
		AuthURL:      authURL,
		ProductosURL: productosURL,
		ClientesURL:  clientesURL,
	}

	handler := facturacion.NewHandler(store, urls)
	mux := http.NewServeMux()
	handler.RegisterRoutes(mux)

	finalHandler := middleware.CORS(mux)

	server := &http.Server{
		Addr:         ":" + port,
		Handler:      finalHandler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		<-sigChan
		log.Println("Apagando servicio de facturación...")
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		server.Shutdown(ctx)
	}()

	log.Printf("Servicio de Facturación escuchando en http://localhost:%s", port)
	if err := server.ListenAndServe(); err != http.ErrServerClosed {
		log.Fatalf("Error en servidor: %v", err)
	}
	log.Println("Servicio de facturación detenido correctamente")
}

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}
