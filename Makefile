# Fintech Multipaís - Makefile
# Comandos para desarrollo, testing y deployment

.PHONY: all build run test clean migrate frontend backend docker k8s help

# Variables
BINARY_NAME=fintech-api
WORKER_NAME=fintech-worker
GO=go
NPM=npm
DOCKER=docker
KUBECTL=kubectl

# Colores para output
GREEN=\033[0;32m
YELLOW=\033[1;33m
NC=\033[0m # No Color

# ============================================
# COMANDOS PRINCIPALES
# ============================================

all: help

help: ## Muestra esta ayuda
	@echo "$(GREEN)Fintech Multipaís - Comandos disponibles:$(NC)"
	@echo ""
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  $(YELLOW)%-20s$(NC) %s\n", $$1, $$2}'

# ============================================
# DESARROLLO
# ============================================

install: install-backend install-frontend ## Instala todas las dependencias

install-backend: ## Instala dependencias del backend
	@echo "$(GREEN)Installing backend dependencies...$(NC)"
	cd backend && $(GO) mod download && $(GO) mod tidy

install-frontend: ## Instala dependencias del frontend
	@echo "$(GREEN)Installing frontend dependencies...$(NC)"
	cd frontend && $(NPM) install

run: ## Ejecuta backend y frontend en desarrollo
	@make -j2 run-backend run-frontend

run-backend: ## Ejecuta solo el backend
	@echo "$(GREEN)Starting backend server...$(NC)"
	cd backend && $(GO) run cmd/api/main.go

run-frontend: ## Ejecuta solo el frontend
	@echo "$(GREEN)Starting frontend server...$(NC)"
	cd frontend && $(NPM) run dev

run-worker: ## Ejecuta el worker de procesamiento
	@echo "$(GREEN)Starting worker...$(NC)"
	cd backend && $(GO) run cmd/worker/main.go

# ============================================
# BUILD
# ============================================

build: build-backend build-frontend ## Compila backend y frontend

build-backend: ## Compila el backend
	@echo "$(GREEN)Building backend...$(NC)"
	cd backend && $(GO) build -o bin/$(BINARY_NAME) cmd/api/main.go
	cd backend && $(GO) build -o bin/$(WORKER_NAME) cmd/worker/main.go

build-frontend: ## Compila el frontend
	@echo "$(GREEN)Building frontend...$(NC)"
	cd frontend && $(NPM) run build

# ============================================
# BASE DE DATOS
# ============================================

migrate: ## Ejecuta las migraciones de base de datos
	@echo "$(GREEN)Running database migrations...$(NC)"
	cd backend && $(GO) run cmd/migrate/main.go up

migrate-down: ## Revierte la última migración
	@echo "$(YELLOW)Rolling back last migration...$(NC)"
	cd backend && $(GO) run cmd/migrate/main.go down

migrate-create: ## Crea una nueva migración (uso: make migrate-create name=nombre)
	@echo "$(GREEN)Creating new migration: $(name)$(NC)"
	@mkdir -p backend/migrations
	@touch backend/migrations/$$(date +%Y%m%d%H%M%S)_$(name).up.sql
	@touch backend/migrations/$$(date +%Y%m%d%H%M%S)_$(name).down.sql

db-seed: ## Carga datos iniciales en la base de datos
	@echo "$(GREEN)Seeding database...$(NC)"
	cd backend && $(GO) run cmd/seed/main.go

# ============================================
# TESTING
# ============================================

test: test-backend test-frontend ## Ejecuta todos los tests

test-backend: ## Ejecuta tests del backend
	@echo "$(GREEN)Running backend tests...$(NC)"
	cd backend && $(GO) test -v -race -cover ./...

test-frontend: ## Ejecuta tests del frontend
	@echo "$(GREEN)Running frontend tests...$(NC)"
	cd frontend && $(NPM) run test

test-coverage: ## Genera reporte de cobertura
	@echo "$(GREEN)Generating coverage report...$(NC)"
	cd backend && $(GO) test -coverprofile=coverage.out ./...
	cd backend && $(GO) tool cover -html=coverage.out -o coverage.html

# ============================================
# LINTING Y FORMATO
# ============================================

lint: lint-backend lint-frontend ## Ejecuta linters

lint-backend: ## Ejecuta linter del backend
	@echo "$(GREEN)Linting backend...$(NC)"
	cd backend && golangci-lint run ./...

lint-frontend: ## Ejecuta linter del frontend
	@echo "$(GREEN)Linting frontend...$(NC)"
	cd frontend && $(NPM) run lint

fmt: ## Formatea el código
	@echo "$(GREEN)Formatting code...$(NC)"
	cd backend && $(GO) fmt ./...
	cd frontend && $(NPM) run format

# ============================================
# DOCKER
# ============================================

docker-build: ## Construye imágenes Docker
	@echo "$(GREEN)Building Docker images...$(NC)"
	$(DOCKER) build -t fintech-api:latest -f docker/Dockerfile.api .
	$(DOCKER) build -t fintech-worker:latest -f docker/Dockerfile.worker .
	$(DOCKER) build -t fintech-frontend:latest -f docker/Dockerfile.frontend .

docker-up: ## Inicia contenedores Docker
	@echo "$(GREEN)Starting Docker containers...$(NC)"
	$(DOCKER) compose up -d

docker-down: ## Detiene contenedores Docker
	@echo "$(YELLOW)Stopping Docker containers...$(NC)"
	$(DOCKER) compose down

docker-logs: ## Muestra logs de Docker
	$(DOCKER) compose logs -f

# ============================================
# KUBERNETES
# ============================================

k8s-deploy: ## Despliega en Kubernetes
	@echo "$(GREEN)Deploying to Kubernetes...$(NC)"
	$(KUBECTL) apply -f k8s/namespace.yaml
	$(KUBECTL) apply -f k8s/configmap.yaml
	$(KUBECTL) apply -f k8s/secrets.yaml
	$(KUBECTL) apply -f k8s/deployments/
	$(KUBECTL) apply -f k8s/services/
	$(KUBECTL) apply -f k8s/ingress.yaml

k8s-delete: ## Elimina deployment de Kubernetes
	@echo "$(YELLOW)Deleting Kubernetes deployment...$(NC)"
	$(KUBECTL) delete -f k8s/

k8s-status: ## Muestra estado del deployment
	$(KUBECTL) get all -n fintech

k8s-logs: ## Muestra logs de los pods
	$(KUBECTL) logs -f -l app=fintech-api -n fintech

# ============================================
# UTILIDADES
# ============================================

clean: ## Limpia archivos generados
	@echo "$(YELLOW)Cleaning generated files...$(NC)"
	rm -rf backend/bin/
	rm -rf frontend/dist/
	rm -rf backend/coverage.*

swagger: ## Genera documentación Swagger
	@echo "$(GREEN)Generating Swagger documentation...$(NC)"
	cd backend && swag init -g cmd/api/main.go -o docs

env-setup: ## Copia archivos de configuración de ejemplo
	@echo "$(GREEN)Setting up environment files...$(NC)"
	cp -n backend/.env.example backend/.env || true
	cp -n frontend/.env.example frontend/.env || true

# ============================================
# DESARROLLO RÁPIDO
# ============================================

dev: env-setup install migrate run ## Setup completo y ejecutar en desarrollo

quick-start: ## Inicio rápido (asume dependencias instaladas)
	@make -j2 run-backend run-frontend

