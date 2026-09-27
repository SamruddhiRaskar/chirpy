package main

import (
	"chirpy/internal/database"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

type Chirp struct {
	Body string `json:"body"`
}
type apiConfig struct {
	fileserverHits atomic.Int32
	db             *database.Queries
	platform       string
}

type createUserRequest struct {
	Email string `json:"email"`
}

type userResponse struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Email     string    `json:"email"`
}

// Middleware: increments the counter for every /app request
func (cfg *apiConfig) middlewareMetricsInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg.fileserverHits.Add(1)
		next.ServeHTTP(w, r)
	})
}

func (cfg *apiConfig) metricsHandler(w http.ResponseWriter, r *http.Request) {
	hits := cfg.fileserverHits.Load()
	html := fmt.Sprintf(`
	<html>
	<body>
	<h1>Welcome, Chirpy Admin</h1>
	<p>Chirpy has been visited %d times!</p>
	</body>
	</html>
		`, hits)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(html))
}

// reset handler
func (cfg *apiConfig) resetHandler(w http.ResponseWriter, r *http.Request) {
	if cfg.platform != "dev" {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}
	cfg.fileserverHits.Store(0)

	err := cfg.db.DeleteAllUsers(r.Context())
	if err != nil {
		http.Error(w, "could not delete users", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Hits reset to 0\n"))
}

// healthz handler
func healthzHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

// validate_chirp handler
func validateChirpHandler(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	var chirp Chirp
	err := json.NewDecoder(r.Body).Decode(&chirp)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if len(chirp.Body) > 140 {
		http.Error(w, "Chirp  is too large", http.StatusBadRequest)
	} else {
		w.Header().Set("Content-Type", "application/json")
		chirp.Body = strings.ReplaceAll(chirp.Body, "kerfuffle", "****")
		chirp.Body = strings.ReplaceAll(chirp.Body, "fornax", "****")
		chirp.Body = strings.ReplaceAll(chirp.Body, "sharbert", "****")

		w.WriteHeader(http.StatusOK)

		json.NewEncoder(w).Encode(map[string]any{
			"cleaned_body": chirp.Body,
			"valid":        true,
		})
	}
}
func (apiCfg *apiConfig) createUserHandler(w http.ResponseWriter, r *http.Request) {

	var req createUserRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		// handle error
		fmt.Println("error in decoding body :", err)
		return
	}
	fmt.Println("decoding passed :", req)

	user, err := apiCfg.db.CreateUser(r.Context(), req.Email)
	if err != nil {
		// handle error
		fmt.Println("error in creating user in database:", err)
		return
	}
	fmt.Println("created user in database:", user)

	resp := userResponse{
		ID:        user.ID,
		Email:     user.Email,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	json.NewEncoder(w).Encode(resp)
}

func main() {
	err := godotenv.Load("./.env")
	if err != nil {
		log.Fatal("Error loading .env file", err)
	}
	dbURL := os.Getenv("DB_URL")
	platform := os.Getenv("PLATFORM")

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatal("Error opening database:", err)
	}

	if err := db.Ping(); err != nil {
		log.Fatal("Error connecting to database:", err)
	}

	fmt.Println("Database connected successfully!")

	dbQueries := database.New(db)

	// Create API config
	apiCfg := &apiConfig{
		db:       dbQueries,
		platform: platform,
	}

	// Create router
	mux := http.NewServeMux()

	fileserver := http.FileServer(http.Dir("."))
	// here dir(.) says that look at the current directory means whats in folder

	appHandler := http.StripPrefix("/app", fileserver)

	mux.Handle(
		"/app/",
		apiCfg.middlewareMetricsInc(appHandler),
	)

	// Other routes
	mux.HandleFunc("GET /api/healthz", healthzHandler)
	mux.HandleFunc("GET /admin/metrics", apiCfg.metricsHandler)
	mux.HandleFunc("POST /admin/reset", apiCfg.resetHandler)
	mux.HandleFunc("POST /api/validate_chirp", validateChirpHandler)
	mux.HandleFunc("POST /api/users", apiCfg.createUserHandler)

	server := http.Server{
		Addr:    ":8080",
		Handler: mux,
	}

	fmt.Println("Listening on port 8080")
	if err := server.ListenAndServe(); err != nil {
		fmt.Println("Error starting server:", err)
	}

	//here it start the server

}
