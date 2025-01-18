# Embedding Proxy Server

A lightweight caching proxy server for embedding APIs that stores embeddings in SQLite, reducing API calls and improving response times for repeated requests.

## Features

- Caches embeddings in SQLite database
- Compatible with OpenAI-compatible embedding APIs (including local alternatives like Ollama)
- Supports multiple embedding models simultaneously
- Simple HTTP API interface
- Zero configuration required for basic usage
- Cross-platform support (Linux, macOS, Windows)

## Quick Start

### Binary Installation

Download the latest binary for your operating system from the [releases page](https://github.com/eja/proxemb/releases).

Run the server:

```bash
./proxemb
```

The server will start with default settings (localhost:35248) and create an SQLite database in the current directory.

### Building from Source

Requirements:
- Go 1.21 or later
- Make (optional)

To compile:

```bash
make
```

Or manually with Go:

```bash
go build -o proxemb
```

## Usage

### Command Line Options

```
Options:

  -api-key string
        OpenAI API key
  -api-url string
        OpenAI API URL (default "http://localhost:11434/v1/")
  -db string
        Path to the SQLite database (default "embeddings.db")
  -log-file string
        Log file path
  -web-host string
        Web server host (default "localhost")
  -web-port string
        Web server port (default "35248")
```

### Example Usage

1. Start the server with default settings:
```bash
./proxemb
```

2. Start with custom API endpoint (e.g., for OpenAI):
```bash
./proxemb -api-url "https://api.openai.com/v1/" -api-key "your-api-key"
```

3. Start with custom port and host:
```bash
./proxemb -web-host "0.0.0.0" -web-port "8080"
```

### API Usage

Send POST requests to the root endpoint with the following JSON structure:

```json
{
    "model": "text-embedding-3-small",
    "input": "Your text to embed"
}
```

Example using curl:

```bash
curl -X POST http://localhost:35248/ \
    -H "Content-Type: application/json" \
    -d '{"model":"text-embedding-3-small","input":"Hello, world!"}'
```

Response format:

```json
{
    "embedding": [0.123, 0.456, ...]
}
```

## Database

The SQLite database is automatically created and contains two tables:
- `models`: Stores model names and their IDs
- `hashes`: Stores the cached embeddings with MD5 hashes of input texts

The database file location can be specified using the `-db` flag.

