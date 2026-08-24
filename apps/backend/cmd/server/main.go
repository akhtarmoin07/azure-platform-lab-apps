package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

type medicine struct {
	ID          int     `json:"id"`
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Price       float64 `json:"price"`
}

type application struct {
	database *sql.DB
	logger   *slog.Logger
}

func main() {
	if len(os.Args) == 3 && os.Args[1] == "healthcheck" {
		if err := runHealthcheck(os.Args[2]); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		logger.Error("DATABASE_URL is required")
		os.Exit(1)
	}

	database, err := sql.Open("pgx", databaseURL)
	if err != nil {
		logger.Error("open database", "error", err)
		os.Exit(1)
	}
	defer database.Close()

	database.SetMaxOpenConns(10)
	database.SetMaxIdleConns(5)
	database.SetConnMaxLifetime(30 * time.Minute)

	startupContext, cancelStartup := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancelStartup()
	if err := initialiseDatabase(startupContext, database); err != nil {
		logger.Error("initialise database", "error", err)
		os.Exit(1)
	}

	app := &application{database: database, logger: logger}
	server := &http.Server{
		Addr:              environmentOrDefault("HTTP_ADDRESS", ":8080"),
		Handler:           app.routes(),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	shutdownSignal, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		logger.Info("backend started", "address", server.Addr)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("serve HTTP", "error", err)
			os.Exit(1)
		}
	}()

	<-shutdownSignal.Done()
	shutdownContext, cancelShutdown := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancelShutdown()
	if err := server.Shutdown(shutdownContext); err != nil {
		logger.Error("graceful shutdown", "error", err)
		os.Exit(1)
	}
	logger.Info("backend stopped")
}

func (app *application) routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health/live", app.live)
	mux.HandleFunc("GET /health/ready", app.ready)
	mux.HandleFunc("GET /api/medicines", app.listMedicines)
	return requestLogger(app.logger, mux)
}

func (app *application) live(response http.ResponseWriter, _ *http.Request) {
	writeJSON(response, http.StatusOK, map[string]string{"status": "alive"})
}

func (app *application) ready(response http.ResponseWriter, request *http.Request) {
	contextWithTimeout, cancel := context.WithTimeout(request.Context(), 2*time.Second)
	defer cancel()
	if err := app.database.PingContext(contextWithTimeout); err != nil {
		writeJSON(response, http.StatusServiceUnavailable, map[string]string{"status": "not-ready"})
		return
	}
	writeJSON(response, http.StatusOK, map[string]string{"status": "ready"})
}

func (app *application) listMedicines(response http.ResponseWriter, request *http.Request) {
	rows, err := app.database.QueryContext(request.Context(), `
		SELECT id, name, description, price
		FROM medicines
		ORDER BY id
	`)
	if err != nil {
		app.logger.Error("query medicines", "error", err)
		writeJSON(response, http.StatusInternalServerError, map[string]string{"error": "query failed"})
		return
	}
	defer rows.Close()

	medicines := make([]medicine, 0)
	for rows.Next() {
		var item medicine
		if err := rows.Scan(&item.ID, &item.Name, &item.Description, &item.Price); err != nil {
			app.logger.Error("scan medicine", "error", err)
			writeJSON(response, http.StatusInternalServerError, map[string]string{"error": "scan failed"})
			return
		}
		medicines = append(medicines, item)
	}
	if err := rows.Err(); err != nil {
		app.logger.Error("iterate medicines", "error", err)
		writeJSON(response, http.StatusInternalServerError, map[string]string{"error": "iteration failed"})
		return
	}
	writeJSON(response, http.StatusOK, medicines)
}

func initialiseDatabase(ctx context.Context, database *sql.DB) error {
	if err := database.PingContext(ctx); err != nil {
		return fmt.Errorf("ping database: %w", err)
	}

	const schema = `
		CREATE TABLE IF NOT EXISTS medicines (
			id SERIAL PRIMARY KEY,
			name TEXT NOT NULL UNIQUE,
			description TEXT NOT NULL,
			price NUMERIC(10, 2) NOT NULL CHECK (price >= 0)
		);
		INSERT INTO medicines (name, description, price) VALUES
			('Vitamin D3', 'Daily vitamin D supplement', 8.99),
			('Saline Nasal Spray', 'Gentle isotonic nasal spray', 5.49),
			('Digital Thermometer', 'Fast-reading digital thermometer', 12.90)
		ON CONFLICT (name) DO NOTHING;
	`
	if _, err := database.ExecContext(ctx, schema); err != nil {
		return fmt.Errorf("apply schema: %w", err)
	}
	return nil
}

func writeJSON(response http.ResponseWriter, status int, value any) {
	response.Header().Set("Content-Type", "application/json")
	response.WriteHeader(status)
	_ = json.NewEncoder(response).Encode(value)
}

func requestLogger(logger *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		started := time.Now()
		next.ServeHTTP(response, request)
		logger.Info("request", "method", request.Method, "path", request.URL.Path, "duration_ms", time.Since(started).Milliseconds())
	})
}

func environmentOrDefault(name, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
}

func runHealthcheck(endpoint string) error {
	client := &http.Client{Timeout: 2 * time.Second}
	response, err := client.Get(endpoint)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("health endpoint returned %s", response.Status)
	}
	return nil
}
