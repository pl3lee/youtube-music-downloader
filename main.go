package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"os/exec"

	"github.com/joho/godotenv"
)

type Config struct {
	Password string
	Port string
}

type ErrorResponse struct {
	Error string `json:"error" example:"Invalid input"`
}

func RespondWithError(w http.ResponseWriter, code int, msg string, err error) {
	if err != nil {
		log.Println(err)
	}
	if code > 499 {
		log.Printf("Responding with 5XX error: %s", msg)
	}
	RespondWithJSON(w, code, ErrorResponse{
		Error: msg,
	})
}

func RespondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	if code == http.StatusNoContent {
		w.WriteHeader(code)
		return
	}
	dat, err := json.Marshal(payload)
	if err != nil {
		log.Printf("Error marshalling JSON: %s", err)
		w.WriteHeader(500)
		return
	}
	w.WriteHeader(code)
	_, err = w.Write(dat)
	if err != nil {
		log.Printf("Write failed: %v", err)
	}
}

func (cfg *Config) authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		clientPass := r.Header.Get("Authorization")
		if cfg.Password == "" {
			next.ServeHTTP(w, r)
		}
		if clientPass != cfg.Password {
			RespondWithError(w, http.StatusUnauthorized, "Unauthorized", nil)
			return
		}
		next.ServeHTTP(w, r)
	})
}

type DownloadRequest struct {
	Links []string `json:"links"`
}

type Result struct {
	Status string `json:"status"`
	Error string `json:"error"`
}

type DownloadResponse struct {
	Results []Result `json:"results"`
}

func (cfg *Config) handlerDownload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		RespondWithError(w, http.StatusMethodNotAllowed, "only POST method allowed", nil)
		return
	}
	// Get youtube music links from body
	var body DownloadRequest
	decoder := json.NewDecoder(r.Body)
	defer r.Body.Close()
	err := decoder.Decode(&body)
	if err != nil {
		RespondWithError(w, http.StatusBadRequest, "cannot decode request body", err)
		return
	}
	if len(body.Links) == 0 {
		RespondWithError(w, http.StatusBadRequest, "no links provided", nil)
		return
	}

	results := []Result{}
	for _, link := range body.Links {
		cmd := exec.Command("gytmdl", "--output-path", "./Music", link)
		log.Printf("Downloading %s...", link)
		_, err := cmd.CombinedOutput()
		status := "success"
		commandError := ""
		if err != nil {
			status = "fail"
			commandError = err.Error()
		}

		results = append(results, Result{
			Status: status,
			Error: commandError,
		})
	}
	RespondWithJSON(w, http.StatusOK, DownloadResponse{
		Results: results,
	})
}


func main() {
	err := godotenv.Load()
	if err != nil {
		log.Println("Warning: Cannot load .env file")
	}
	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}
	pwd := os.Getenv("PASSWORD")
	if pwd == "" {
		log.Println("Warning: No password set")
	}
	cfg := Config{
		Port: port,
		Password: pwd,
	}

	mux := http.NewServeMux()
	mux.Handle("/api/download", cfg.authMiddleware(http.HandlerFunc(cfg.handlerDownload)))
	log.Printf("Server listening on :%s\n", port)
	http.ListenAndServe(":" + port, mux)
	
}
