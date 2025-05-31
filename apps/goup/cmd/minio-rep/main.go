package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Core structures inspired by MinIO's architecture
type ObjectInfo struct {
	Name         string            `json:"name"`
	Size         int64             `json:"size"`
	ModTime      time.Time         `json:"modTime"`
	ETag         string            `json:"etag"`
	UserDefined  map[string]string `json:"userDefined,omitempty"`
	VersionID    string            `json:"versionId,omitempty"`
	DeleteMarker bool              `json:"deleteMarker,omitempty"`
}

type MinioServer struct {
	dataDir string
	port    string
}

func NewMinioServer(dataDir, port string) *MinioServer {
	return &MinioServer{
		dataDir: dataDir,
		port:    port,
	}
}

// Core API handlers similar to MinIO's object-handlers.go
func (s *MinioServer) putObjectHandler(w http.ResponseWriter, r *http.Request) {
	bucket := strings.Split(r.URL.Path, "/")[1]
	object := strings.Join(strings.Split(r.URL.Path, "/")[2:], "/")

	if bucket == "" || object == "" {
		http.Error(w, "Invalid bucket or object name", http.StatusBadRequest)
		return
	}

	bucketDir := filepath.Join(s.dataDir, bucket)
	if err := os.MkdirAll(bucketDir, 0755); err != nil {
		http.Error(w, "Failed to create bucket", http.StatusInternalServerError)
		return
	}

	objectPath := filepath.Join(bucketDir, object)
	objectDir := filepath.Dir(objectPath)
	if err := os.MkdirAll(objectDir, 0755); err != nil {
		http.Error(w, "Failed to create object directory", http.StatusInternalServerError)
		return
	}

	file, err := os.Create(objectPath)
	if err != nil {
		http.Error(w, "Failed to create object", http.StatusInternalServerError)
		return
	}
	defer file.Close()

	_, err = io.Copy(file, r.Body)
	if err != nil {
		http.Error(w, "Failed to write object", http.StatusInternalServerError)
		return
	}

	w.Header().Set("ETag", fmt.Sprintf("\"%x\"", time.Now().Unix()))
	w.WriteHeader(http.StatusOK)
}

func (s *MinioServer) getObjectHandler(w http.ResponseWriter, r *http.Request) {
	bucket := strings.Split(r.URL.Path, "/")[1]
	object := strings.Join(strings.Split(r.URL.Path, "/")[2:], "/")

	objectPath := filepath.Join(s.dataDir, bucket, object)

	file, err := os.Open(objectPath)
	if err != nil {
		if os.IsNotExist(err) {
			http.Error(w, "Object not found", http.StatusNotFound)
		} else {
			http.Error(w, "Failed to read object", http.StatusInternalServerError)
		}
		return
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		http.Error(w, "Failed to stat object", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Length", fmt.Sprintf("%d", stat.Size()))
	w.Header().Set("Last-Modified", stat.ModTime().UTC().Format(http.TimeFormat))
	w.Header().Set("ETag", fmt.Sprintf("\"%x\"", stat.ModTime().Unix()))

	io.Copy(w, file)
}

func (s *MinioServer) listObjectsHandler(w http.ResponseWriter, r *http.Request) {
	bucket := strings.Split(r.URL.Path, "/")[1]
	prefix := r.URL.Query().Get("prefix")

	bucketDir := filepath.Join(s.dataDir, bucket)

	var objects []ObjectInfo

	err := filepath.Walk(bucketDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		relPath, err := filepath.Rel(bucketDir, path)
		if err != nil {
			return err
		}

		relPath = filepath.ToSlash(relPath)

		if prefix != "" && !strings.HasPrefix(relPath, prefix) {
			return nil
		}

		objects = append(objects, ObjectInfo{
			Name:    relPath,
			Size:    info.Size(),
			ModTime: info.ModTime(),
			ETag:    fmt.Sprintf("\"%x\"", info.ModTime().Unix()),
		})

		return nil
	})

	if err != nil && !os.IsNotExist(err) {
		http.Error(w, "Failed to list objects", http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"Name":     bucket,
		"Contents": objects,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (s *MinioServer) deleteObjectHandler(w http.ResponseWriter, r *http.Request) {
	bucket := strings.Split(r.URL.Path, "/")[1]
	object := strings.Join(strings.Split(r.URL.Path, "/")[2:], "/")

	objectPath := filepath.Join(s.dataDir, bucket, object)

	err := os.Remove(objectPath)
	if err != nil {
		if os.IsNotExist(err) {
			http.Error(w, "Object not found", http.StatusNotFound)
		} else {
			http.Error(w, "Failed to delete object", http.StatusInternalServerError)
		}
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// Router similar to MinIO's routers.go
func (s *MinioServer) setupRoutes() *http.ServeMux {
	mux := http.NewServeMux()

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPut:
			s.putObjectHandler(w, r)
		case http.MethodGet:
			if strings.Contains(r.URL.RawQuery, "list-type=2") || r.URL.Path == "/" {
				s.listObjectsHandler(w, r)
			} else {
				s.getObjectHandler(w, r)
			}
		case http.MethodDelete:
			s.deleteObjectHandler(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	return mux
}

func (s *MinioServer) Start(ctx context.Context) error {
	if err := os.MkdirAll(s.dataDir, 0755); err != nil {
		return fmt.Errorf("failed to create data directory: %v", err)
	}

	mux := s.setupRoutes()
	server := &http.Server{
		Addr:    ":" + s.port,
		Handler: mux,
	}

	log.Printf("Starting MinIO-like server on port %s, data directory: %s", s.port, s.dataDir)

	go func() {
		<-ctx.Done()
		log.Println("Shutting down server...")
		server.Shutdown(context.Background())
	}()

	return server.ListenAndServe()
}

func main() {
	dataDir := "./data"
	if len(os.Args) > 1 {
		dataDir = os.Args[1]
	}

	port := "9000"
	if len(os.Args) > 2 {
		port = os.Args[2]
	}

	server := NewMinioServer(dataDir, port)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := server.Start(ctx); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Server failed: %v", err)
	}
}
