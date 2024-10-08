package main

import (
	"log"
	"os"

	"github.com/Abraxas-365/neurons/internal/classroom"
	"github.com/Abraxas-365/neurons/internal/user"
	"github.com/Abraxas-365/toolkit/pkg/errors"
	"github.com/Abraxas-365/toolkit/pkg/lucia"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

func main() {
	// Set up database connection
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL environment variable is not set")
	}

	// Append sslmode=disable to the connection string
	dbURL += "?sslmode=disable"

	db, err := sqlx.Connect("postgres", dbURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Test database connection
	err = db.Ping()
	if err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}

	// Initialize repositories
	userRepo := user.NewPostgresRepository(db)
	classroomRepo := classroom.NewPostgresRepository(db)

	// Initialize services
	userService := user.NewService(userRepo)
	classroomService := classroom.NewService(userService, classroomRepo)

	luciaRepo := lucia.NewPostgresRepository(db)
	luciaService := lucia.NewService(luciaRepo)

	// Initialize handlers
	classroomHandler := classroom.NewHandler(classroomService)
	userHandler := user.NewHandler(userService)

	// Create Fiber app
	app := fiber.New(fiber.Config{
		ErrorHandler: errors.ErrorHandler,
	})

	// Configure CORS
	app.Use(cors.New(cors.Config{
		AllowOrigins:     "http://localhost:5173", // Update this to match your SvelteKit dev server
		AllowMethods:     "GET,POST,PUT,DELETE,OPTIONS",
		AllowHeaders:     "Origin,Content-Type,Accept,Authorization",
		AllowCredentials: true,
	}))

	// Use middlewares
	app.Use(recover.New())
	app.Use(logger.New())

	// Set up authentication middleware
	app.Use(lucia.SessionMiddleware(luciaService))

	classroomHandler.RegisterRoutes(app)
	userHandler.RegisterRoutes(app)

	// Start server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("Server starting on port %s", port)
	log.Fatal(app.Listen(":" + port))
}
