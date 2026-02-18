# APIM Core: The Modern API Gateway for Navante

Welcome to the official documentation and "How-To" site for APIM Core. This is a lightweight, high-performance API Gateway built in Go, designed to manage, meter, and secure your microservices with ease.

---

## 🚀 Quick Navigation

Explore our detailed guides to get up and running:

- [**Getting Started**](docs/getting-started.md) - Install and run APIM Core in minutes.
- [**Configuration Guide**](docs/configuration.md) - Learn how to use the YAML-driven setup and Hot-Reloading.
- [**Architecture Overview**](docs/architecture.md) - Understand the internal design and components.

---

## ✨ Core Features

### 🛡️ Secure & Scalable Gateway
A robust reverse proxy that routes requests to your backends (EduCore, IntegraCore, etc.) based on clean URL paths.

### 📊 Real-time Metering
Automatic tracking of requests per API, method, status, and latency. Fully integrated with Prometheus for advanced monitoring.

### 🔄 YAML-Driven & Decoupled
Manage your entire infrastructure through a single, version-controlled file. No database or complex frontend required for core operations.

### ⚡ Dynamic Hot-Reload
Change your configuration on the fly. APIM Core watches for file updates and applies them instantly without dropping a single connection.

### 🌐 Integrated Developer Portal
A built-in portal for your developers to explore API documentation and monitor their own usage metrics.

---

## 🛠️ Management & API Reference

| Component | Port | Key Endpoints |
|-----------|------|---------------|
| **Gateway** | `:8080` | `/educore/*`, `/identity/*` |
| **Server** | `:8081` | `/metrics`, `/devportal`, `/api/admin/*` |

### Key API Endpoints (Admin)
- `GET /api/admin/products`: List all configured products.
- `GET /api/admin/usage`: Retrieve real-time usage statistics.
- `GET /health`: System health status.

---

## 🏗️ Technical Stack
- **Language:** Go 1.22+
- **Monitoring:** Prometheus
- **Deployment:** Docker & Docker Compose
- **Configuration:** YAML (with Hot-Reload support)

---

## 🤝 Contributing
Please read our [Architecture Guide](docs/architecture.md) before submitting pull requests to ensure alignment with the project's design principles.

---

*© 2024 Navante Solutions. Powered by Go.*
