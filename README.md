# APIM Core 🚀

**APIM Core** is a lightweight, high-performance, and Go-native API Management solution designed for modern distributed architectures. It provides a robust Gateway with a built-in interactive Management Hub (TUI), enabling real-time monitoring and control of your API ecosystem.

Inspired by industry leaders like Apache APISIX, APIM Core brings a modular middleware engine and a powerful YAML-driven configuration system to your local environment.

---

## ✨ Key Features

- 🛡️ **Advanced Security**: Built-in IP Blacklisting, CIDR blocking, and Anti-DDoS rate limiting.
- 🌍 **Geo-fencing**: Regional access control based on GeoIP resolution (mocked in core, production-ready logic).
- ⛓️ **Middleware Engine**: Flexible request processing pipeline for easy expansion.
- 🚦 **Multi-tenancy**: First-class support for tenant-based routing and metrics.
- 🎮 **Turbo Management Hub (TUI)**: A comprehensive console for real-time monitoring, traffic analysis, and administration.
- 📦 **Plug-and-Play**: Single binary with no external dependencies (Redis/DB optional for persistence).

---

## 🛠️ Getting Started

### Prerequisites
- Go 1.22+

### Quick Start
1. **Clone and Build**:
   ```bash
   git clone https://github.com/navante-solutions/apimcore.git
   cd apimcore
   go build ./cmd/apim
   ```

2. **Run with Default Config**:
   ```bash
   ./apim --config config.yaml --tui
   ```

3. **Explore the TUI**:
   Press `F3` to open the Navigation Menu and explore Dashboard, Traffic, Admin, and Security views.

---

## 🎮 The TUI Management Hub

APIM Core features a state-of-the-art terminal interface (TUI) for management:

- **Dashboard (F3 Menu)**: System vitals, uptime, and real-time event logs.
- **Traffic Monitor (F4)**: Wireshark-style request inspector with GeoIP flags and security highlighting.
- **Administration (F5)**: Live view of Products, API Definitions, and Subscriptions.
- **Security Control (F6)**: Interactive management of Blacklists and Geo-fencing policies.
- **System Health (F7)**: Integrated health checks and internal metrics.

---

## ⚙️ Configuration scenarios

We provide a library of configuration examples in the `/examples` directory:

- [basic.yaml](examples/basic.yaml): Simple one-product, one-API setup.
- [security.yaml](examples/security.yaml): Strict rate limits and IP protection.
- [multi-tenant.yaml](examples/multi_tenant.yaml): Complex Enterprise-grade multi-tenant configuration.
- [geo-fencing.yaml](examples/geo_fencing.yaml): Regional access control (e.g., EU-only vs Global).

---

## 🤝 Contributing

We welcome contributions! Please feel free to submit Pull Requests or open issues for feature requests and bug reports.

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
