// Servidor estático del frontend - Puerto 3000
// Sirve la carpeta frontend/ como archivos estáticos.
package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	port := getEnv("PORT", "3000")
	frontendDir := getEnv("FRONTEND_DIR", "frontend")

	log.Printf("Servidor web iniciando en puerto %s", port)
	log.Printf("   Sirviendo archivos desde: %s", frontendDir)

	fs := http.FileServer(http.Dir(frontendDir))
	mux := http.NewServeMux()
	mux.Handle("/", fs)

	server := &http.Server{
		Addr:         ":" + port,
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		<-sigChan
		log.Println("Apagando servidor web...")
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		server.Shutdown(ctx)
	}()

	log.Printf("Servidor web escuchando en http://localhost:%s", port)
	if err := server.ListenAndServe(); err != http.ErrServerClosed {
		log.Fatalf("Error en servidor web: %v", err)
	}
	log.Println("Servidor web detenido correctamente")
}

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}
