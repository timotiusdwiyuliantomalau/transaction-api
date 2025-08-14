# Transaction API

A simple backend system for managing transactions built using Golang, Gin Gonic, GORM, and MySQL. It provides a RESTful API for CRUD transaction operations and a summary dashboard for data analysis.

## 🚀 Set Up

### 1. Clone Repository

```bash
git clone <repository-url>
cd transaction-api
```

### 2. Environment Setup

Copy file environment:

```bash
cp .env.example .env
```

Edit `.env` sesuai dengan konfigurasi database `docker-compose.yaml`

# Server Configuration

SERVER_PORT=8080
GIN_MODE=debug

# Log Configuration

LOG_LEVEL=info

````

### 3. Database Setup (Menggunakan Docker)

Gunakan Docker Compose
```bash
docker-compose up -d
````

### 4. Install Dependencies

```bash
go mod tidy
```

### 5. Run Application

```bash
go run cmd/server/main.go
```

Server akan berjalan di `http://localhost:8080`

## 🔍 Testing

Jalankan unit tests:

```bash
go test ./... -v
```

Jalankan tests dengan coverage:

```bash
go test ./... -cover
```
