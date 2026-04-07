package main

import (
	"database/sql"
	"fmt"
	"html/template"
	"log"
	"os"

	"github.com/gin-gonic/gin"
	_ "github.com/mattn/go-sqlite3"

	"github.com/wouter/voting-with-draw/internal/config"
	dbsqlc "github.com/wouter/voting-with-draw/internal/db/sqlc"
	"github.com/wouter/voting-with-draw/internal/handlers"
	ws "github.com/wouter/voting-with-draw/internal/websocket"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	cfgPath := "config.yaml"
	if p := os.Getenv("CONFIG_PATH"); p != "" {
		cfgPath = p
	}

	cfg, err := config.Load(cfgPath)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	dbPath := "data/votes.db"
	if p := os.Getenv("DB_PATH"); p != "" {
		dbPath = p
	}

	db, err := sql.Open("sqlite3", dbPath+"?_journal_mode=WAL&_busy_timeout=5000&_foreign_keys=on")
	if err != nil {
		return fmt.Errorf("opening database: %w", err)
	}
	defer func() { _ = db.Close() }()

	if err := runMigrations(db); err != nil {
		return fmt.Errorf("running migrations: %w", err)
	}

	queries := dbsqlc.New(db)
	hub := ws.NewHub()

	// Handlers
	voteHandler := handlers.NewVoteHandler(cfg, queries, hub)
	resultsHandler := handlers.NewResultsHandler(cfg, queries, hub)
	adminHandler := handlers.NewAdminHandler(cfg, db, queries, hub)

	// Gin setup
	r := gin.Default()

	// Custom template functions
	r.SetFuncMap(template.FuncMap{
		"percentage": func(count, total int64) float64 {
			if total == 0 {
				return 0
			}
			return float64(count) / float64(total) * 100
		},
	})

	r.LoadHTMLGlob("templates/*")
	r.Static("/static", "./static")
	r.Static("/photos", "./photos")

	// Public routes
	r.GET("/", voteHandler.VotePage)
	r.GET("/thankyou", voteHandler.ThankYouPage)
	r.POST("/api/vote", voteHandler.SubmitVote)
	r.GET("/results", resultsHandler.ResultsPage)
	r.GET("/ws/results", resultsHandler.WebSocket)

	// Admin routes
	admin := r.Group("/admin", adminHandler.AdminAuth())
	{
		admin.GET("", adminHandler.AdminPage)
		admin.POST("/clear", adminHandler.ClearVotes)
		admin.POST("/draw", adminHandler.RunDraw)
	}

	addr := fmt.Sprintf(":%d", cfg.Server.Port)
	log.Printf("Starting server on %s", addr)
	log.Printf("Exhibition: %s (%d photos)", cfg.Exhibition.Title, len(cfg.Photos))

	return r.Run(addr)
}

func runMigrations(db *sql.DB) error {
	migration, err := os.ReadFile("internal/db/migrations/001_init.sql")
	if err != nil {
		return fmt.Errorf("reading migration: %w", err)
	}
	if _, err := db.Exec(string(migration)); err != nil {
		return fmt.Errorf("running migration: %w", err)
	}
	log.Println("Database migrations applied")
	return nil
}
