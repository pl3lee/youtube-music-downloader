package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"sync"

	"github.com/google/uuid"
	"github.com/joho/godotenv"
)

type Config struct {
	Password string
	Port     string
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
			return
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
	Link   string `json:"link"`
	Status string `json:"status"`
	Error  string `json:"error,omitempty"`
}

type DownloadResponse struct {
	Results []Result `json:"results"`
}

type TaskCreationResponse struct {
	TaskID string `json:"task_id"`
}

type DownloadTask struct {
	ID string
	Links []string
	Updates chan Result // Channel to send updates for each link
	Done chan bool // Channel to signal task completion
	AuthHeader string // Auth header for SSD endpoint
}

var (
	activeTasks = make(map[string]*DownloadTask)
	activeTasksMutex = &sync.Mutex{}
)

func (cfg *Config) processDownloadTask(task *DownloadTask) {
	// Function calls to close channels and signal completion
	defer func() {
		task.Done <- true // Signal completion
		close(task.Updates)
		close(task.Done)
		activeTasksMutex.Lock()
		delete(activeTasks, task.ID)
		activeTasksMutex.Unlock()
		log.Printf("Task %s completed and cleaned up.", task.ID)
	}()

	// Ensure the ./Music directory exists
	musicDir := "./Music"
	if _, err := os.Stat(musicDir); os.IsNotExist(err) {
		log.Printf("Music directory %s does not exist, creating it.", musicDir)
		err = os.MkdirAll(musicDir, 0755) // Create directory with appropriate permissions
		if err != nil {
			log.Printf("Error creating music directory")
			// Send general error for all links
			for _, link := range task.Links {
				task.Updates <- Result{
					Link: link,
					Status: "fail",
					Error: "could not create music directory: " + err.Error(),
				}
			}
			return
		}
	}

	for _, link := range task.Links {
		// Process each link
		cmd := exec.Command("gytmdl", "--output-path", musicDir, link)
		log.Printf("Task %s: Downloading %s...", task.ID, link)
		output, err := cmd.CombinedOutput()

		// Update result
		result := Result{Link: link}
		if err != nil {
			result.Status = "fail"
			result.Error = err.Error() + " | Output: " + string(output)
			log.Printf("Task %s: CLI Error for link %s: %s\nCLI Output: %s", task.ID, link, err.Error(), string(output))
		} else {
			result.Status ="success"
			log.Printf("Task %s: Successfully downloaded %s. Output: %s", task.ID, link, string(output))
		}

		// Send result to channel
		task.Updates <- result
	}
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

	// Create new task
	taskID := uuid.New().String()
	task := &DownloadTask {
		ID: taskID,
		Links: body.Links,
		Updates: make(chan Result),
		Done: make(chan bool),
		AuthHeader: r.Header.Get("Authorization"),
	}

	// Store in activeTasks
	activeTasksMutex.Lock()
	activeTasks[taskID] = task
	activeTasksMutex.Unlock()

	go cfg.processDownloadTask(task)
	log.Printf("Task %s created for %d links.", taskID, len(body.Links))
	RespondWithJSON(w, http.StatusAccepted, TaskCreationResponse{TaskID: taskID})
}


func (cfg *Config) handlerDownloadStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		RespondWithError(w, http.StatusMethodNotAllowed, "only GET method allowed for status", nil)
		return
	}

	taskID := strings.TrimPrefix(r.URL.Path, "/api/download/status/")
	if taskID == "" {
		RespondWithError(w, http.StatusBadRequest, "Task ID missing in URL path", nil)
		return
	}

	activeTasksMutex.Lock()
	task, ok := activeTasks[taskID]
	activeTasksMutex.Unlock()

	if !ok {
		RespondWithError(w, http.StatusNotFound, "Task ID not found or already completed", nil)
		return
	}

	// SSE endpoint should be protected as well
	// Check if the task was created with an auth header that matches the app's password
	if cfg.Password != "" && task.AuthHeader != cfg.Password {
		RespondWithError(w, http.StatusUnauthorized, "Unauthorized for task status", nil)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	flusher, ok := w.(http.Flusher)
	if !ok {
		RespondWithError(w, http.StatusInternalServerError, "Streaming unsupported", nil)
		return
	}

	log.Printf("SSE connection established for task %s", taskID)
	fmt.Fprintf(w, ": connection established for task %s\n\n", taskID)
	flusher.Flush()

	for {
		select {
		case update, ok := <-task.Updates:
			if !ok {
				// Channel closed, task processing loop finished before Done signal
				// This should not happen if Done is always sent
				log.Printf("Task %s: Updates channel closed unexpectedly.", task.ID)
				fmt.Fprintf(w, "event: error\ndata: {\"error\": \"Updates channel closed unexpectedly on server.\"}\n\n")
				flusher.Flush()
				return
			}
			jsonData, err := json.Marshal(update)
			if err != nil {
				log.Printf("Task %s: Error marshalling update to JSON: %v", task.ID, err)
				fmt.Fprintf(w, "event: error\ndata:{\"error\": \"Updates channel closed unexpectedly on server.\"}\n\n")
				flusher.Flush()
				continue
			}
			log.Printf("Task %s: Sending update: %s", task.ID, string(jsonData))
			fmt.Fprintf(w, "data: %s\n\n", jsonData)
			flusher.Flush()
		case <- task.Done:
			log.Printf("Task %s: All items processed. Sending completion event.", task.ID)
			fmt.Fprintf(w, "event: complete\ndata: {\"message\": \"Task completed\"}\n\n")
			flusher.Flush()
			return
		
		case <- r.Context().Done():
			log.Printf("Task %s: Client disconnected.", task.ID)
			// No need to delete task here, processDownloadTask defer will handle it.
			return
		}
	}
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
		Port:     port,
		Password: pwd,
	}

	mux := http.NewServeMux()
	mux.Handle("/api/download", cfg.authMiddleware(http.HandlerFunc(cfg.handlerDownload)))
	mux.Handle("/api/download/status/", http.HandlerFunc(cfg.handlerDownloadStatus))

	uiFileServer := http.FileServer(http.Dir("ui"))
	mux.Handle("/", uiFileServer)
	log.Printf("Server listening on :%s\n", port)
	err = http.ListenAndServe(":"+port, mux)
	if err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
