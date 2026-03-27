package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/guilletrejo/sachaweb/internal/config"
	"github.com/guilletrejo/sachaweb/internal/handler"
	"github.com/guilletrejo/sachaweb/internal/model"
	"github.com/guilletrejo/sachaweb/internal/repository"
	"github.com/guilletrejo/sachaweb/internal/service"
)

func main() {
	// Step 1: Load configuration.
	cfg := config.Load()

	// Step 2: Create the REPOSITORY layer (data storage).
	//
	// We seed it with sample products so there's data to work with.
	// In Phase 3, this becomes a PostgreSQL connection instead.
	sampleProducts := []model.Product{
		{ID: "1", Name: "Mechanical Keyboard", Description: "RGB mechanical keyboard with Cherry MX switches", Price: 8999, Category: "electronics"},
		{ID: "2", Name: "Go Programming Book", Description: "The Go Programming Language by Donovan & Kernighan", Price: 3495, Category: "books"},
		{ID: "3", Name: "USB-C Hub", Description: "7-in-1 USB-C hub with HDMI, USB 3.0, and SD card reader", Price: 4599, Category: "electronics"},
	}
	productRepo := repository.NewMemoryProductRepo(sampleProducts)

	// Step 3: Create the SERVICE layer (business logic).
	//
	// The service receives the repository through its constructor.
	// This is dependency injection — no global variables, no magic.
	// The wiring happens here in main.go, and ONLY here.
	productService := service.NewProductService(productRepo)

	// Step 4: Create the HANDLER layer (HTTP interface).
	//
	// The handler receives the service through its constructor.
	// Notice the chain: handler → service → repository.
	// Each layer only knows about the one directly below it.
	productHandler := handler.NewProductHandler(productService)

	// Step 5: Create router and register routes.
	mux := http.NewServeMux()

	// Health check (unchanged from Phase 1).
	mux.HandleFunc("GET /health", handler.HandleHealth())

	// Product CRUD routes.
	// Each HTTP method + path maps to a specific operation:
	mux.HandleFunc("GET /products", productHandler.HandleList())       // List all
	mux.HandleFunc("GET /products/{id}", productHandler.HandleGet())   // Get one
	mux.HandleFunc("POST /products", productHandler.HandleCreate())    // Create
	mux.HandleFunc("PUT /products/{id}", productHandler.HandleUpdate()) // Update
	mux.HandleFunc("DELETE /products/{id}", productHandler.HandleDelete()) // Delete

	// Step 6: Start server.
	addr := fmt.Sprintf(":%s", cfg.Port)
	log.Printf("Starting server on %s", addr)
	log.Printf("Endpoints:")
	log.Printf("  GET    /health")
	log.Printf("  GET    /products")
	log.Printf("  GET    /products/{id}")
	log.Printf("  POST   /products")
	log.Printf("  PUT    /products/{id}")
	log.Printf("  DELETE /products/{id}")

	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatal(err)
	}
}
