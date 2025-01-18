// Copyright (C) 2025 by Ubaldo Porcheddu <ubaldo@eja.it>

package main

import (
	"context"
	"crypto/md5"
	"database/sql"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math"
	"net/http"
	"os"

	_ "github.com/mattn/go-sqlite3"
	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
	"github.com/openai/openai-go/shared"
)

const Version = "1.1.0"

var (
	dbPath  string
	apiURL  string
	apiKey  string
	logFile string
	webHost string
	webPort string
	db      *sql.DB
	models  map[string]int
)

type EmbeddingRequest struct {
	Model string `json:"model"`
	Input string `json:"input"`
}

type EmbeddingResponse struct {
	Embedding []float32 `json:"embedding"`
}

func initDB() {
	var err error
	db, err = sql.Open("sqlite3", dbPath)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}

	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS models (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			model TEXT UNIQUE
		);
		CREATE TABLE IF NOT EXISTS hashes (
			hash TEXT,
			model INTEGER,
			embedding BLOB,
			PRIMARY KEY (hash, model),
			FOREIGN KEY(model) REFERENCES models(id)
		);
	`)
	if err != nil {
		log.Fatalf("Failed to create tables: %v", err)
	}

	rows, err := db.Query("SELECT id, model FROM models")
	if err != nil {
		log.Fatalf("Failed to query models: %v", err)
	}
	defer rows.Close()

	models = make(map[string]int)
	for rows.Next() {
		var id int
		var model string
		if err := rows.Scan(&id, &model); err != nil {
			log.Fatalf("Failed to scan model row: %v", err)
		}
		models[model] = id
	}
}

func getModelID(model string) int {
	if id, exists := models[model]; exists {
		return id
	}

	result, err := db.Exec("INSERT INTO models (model) VALUES (?)", model)
	if err != nil {
		log.Fatalf("Failed to insert model: %v", err)
	}

	modelID, err := result.LastInsertId()
	if err != nil {
		log.Fatalf("Failed to get last insert ID: %v", err)
	}

	models[model] = int(modelID)
	return int(modelID)
}

func getHash(text string) string {
	hash := md5.Sum([]byte(text))
	hashStr := hex.EncodeToString(hash[:])
	return hashStr
}

func BlobToFloat32(bytes []byte) []float32 {
	if len(bytes)%4 != 0 {
		panic("input byte slice length must be a multiple of 4")
	}

	float32s := make([]float32, 0, len(bytes)/4)
	for i := 0; i < len(bytes); i += 4 {
		bits := binary.LittleEndian.Uint32(bytes[i : i+4])
		float32s = append(float32s, math.Float32frombits(bits))
	}

	return float32s
}

func Float32ToBlob(values []float32) []byte {
	bytes := make([]byte, 4*len(values))

	for i, value := range values {
		bits := math.Float32bits(value)
		binary.LittleEndian.PutUint32(bytes[4*i:4*(i+1)], bits)
	}

	return bytes
}

func getCachedEmbedding(hash string, modelID int) ([]float32, bool) {
	var embeddingBlob []byte
	err := db.QueryRow("SELECT embedding FROM hashes WHERE hash = ? AND model = ?", hash, modelID).Scan(&embeddingBlob)
	if err == sql.ErrNoRows {
		return nil, false
	} else if err != nil {
		log.Fatalf("Failed to query hashes: %v", err)
	}

	return BlobToFloat32(embeddingBlob), true
}

func cacheEmbedding(hash string, modelID int, embedding []float32) {
	embeddingBlob := Float32ToBlob(embedding)
	_, err := db.Exec("INSERT INTO hashes (hash, model, embedding) VALUES (?, ?, ?)", hash, modelID, embeddingBlob)
	if err != nil {
		log.Fatalf("Failed to insert hash: %v", err)
	}
}

func getOpenAIEmbedding(model, input string) ([]float32, error) {
	client := openai.NewClient(
		option.WithAPIKey(apiKey),
		option.WithBaseURL(apiURL),
	)

	response, err := client.Embeddings.New(context.TODO(), openai.EmbeddingNewParams{
		Model:          openai.F(model),
		Input:          openai.F[openai.EmbeddingNewParamsInputUnion](shared.UnionString(input)),
		EncodingFormat: openai.F(openai.EmbeddingNewParamsEncodingFormatFloat),
	})

	if err != nil {
		return nil, fmt.Errorf("failed to create embeddings: %v", err)
	}

	var output []float32
	for _, embedding := range response.Data {
		for _, value := range embedding.Embedding {
			output = append(output, float32(value))
		}
	}

	return output, nil
}

func handleRequest(w http.ResponseWriter, r *http.Request) {
	var req EmbeddingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	modelID := getModelID(req.Model)
	hash := getHash(req.Input)

	if embedding, exists := getCachedEmbedding(hash, modelID); exists {
		json.NewEncoder(w).Encode(EmbeddingResponse{Embedding: embedding})
		return
	}

	embedding, err := getOpenAIEmbedding(req.Model, req.Input)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get embeddings: %v", err), http.StatusInternalServerError)
		return
	}

	cacheEmbedding(hash, modelID, embedding)

	json.NewEncoder(w).Encode(EmbeddingResponse{Embedding: embedding})
}

func main() {
	flag.StringVar(&dbPath, "db", "embeddings.db", "Path to the SQLite database")
	flag.StringVar(&apiURL, "api-url", "http://localhost:11434/v1/", "OpenAI API URL")
	flag.StringVar(&apiKey, "api-key", "", "OpenAI API key")
	flag.StringVar(&logFile, "log-file", "", "Log file path")
	flag.StringVar(&webHost, "web-host", "localhost", "Web server host")
	flag.StringVar(&webPort, "web-port", "35248", "Web server port")

	flag.Usage = func() {
		fmt.Println("Copyright:", "2025 by Ubaldo Porcheddu <ubaldo@eja.it>")
		fmt.Println("Version:", Version)
		fmt.Printf("Usage: %s [options]\n", os.Args[0])
		fmt.Println("Options:\n")
		flag.PrintDefaults()
		fmt.Println()
	}

	flag.Parse()

	if logFile != "" {
		file, err := os.OpenFile(logFile, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error opening log file: %v\n", err)
		}
		log.SetOutput(file)
	}

	initDB()

	http.HandleFunc("/", handleRequest)

	addr := fmt.Sprintf("%s:%s", webHost, webPort)
	log.Printf("Starting embedding proxy server on %s\n", addr)
	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
