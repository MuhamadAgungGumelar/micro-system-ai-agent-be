# Micro-System AI Agent Platform

**Multi-Vertical SaaS Framework** - A modular AI agent platform designed as a foundation for building industry-specific vertical SaaS solutions.

[![Go Version](https://img.shields.io/badge/Go-1.25-blue.svg)](https://golang.org)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)

---

## 🎯 Platform Vision

Build AI-powered automation for multiple industries using a **single, reusable platform**:

- **UMKM** (Small & Medium Business) - Inventory, POS, Accounting
- **Pharmacy** - Drug database, Prescriptions, Compliance
- **Manufacturing** - MRP, Supply Chain, Quality Control
- **More verticals** - Easily extensible to new industries

### Key Principles

✅ **Reusability** - Core services shared across all verticals
✅ **Modularity** - Each vertical developed independently
✅ **Scalability** - Horizontal scaling architecture
✅ **Multi-Tenancy** - One instance serves multiple clients

---

## 🚀 Features

### Core Platform (Reusable Across ALL Verticals)

#### 🤖 **Multi-LLM Support**
- OpenAI (GPT-4, GPT-4o-mini)
- Google Gemini (1.5 Flash, 1.5 Pro)
- Groq (Llama, Mixtral)
- DeepSeek
- Switchable via environment variable

#### 💬 **WhatsApp Integration**
- WAHA provider for WhatsApp Business API
- QR code authentication
- Send/receive messages & media
- Webhook support
- Session management

#### 📸 **OCR Engine with AI**
- Google Cloud Vision / OCR.space / Tesseract
- LLM-enhanced parsing (85-95% accuracy)
- Receipt & document scanning
- Automatic data extraction

#### 🔐 **Authentication System**
- JWT with refresh tokens (2 hour access, 7 day refresh)
- Email/Password authentication
- Google OAuth ready
- Role-based access control (RBAC)
- Multi-tenant isolation

#### 📂 **File Upload Service**
- Multi-provider: Local / Cloudinary / AWS S3
- Switchable via configuration
- Image transformations (resize, crop)
- CDN support

#### 📊 **Vector Database (NEW!)**
- Qdrant integration (Cloud + Self-hosted)
- Semantic search for knowledge base
- OpenAI embeddings (text-embedding-3-small/large)
- RAG (Retrieval-Augmented Generation) ready

#### ⚙️ **Workflow Automation**
- Trigger-based automation (events, scheduled, manual)
- Condition evaluation (AND/OR logic)
- Multi-action support (WhatsApp, DB, API, LLM)
- Cron scheduling for time-based workflows
- Execution logging

#### 💳 **Payment Gateway**
- Manual confirmation mode
- Midtrans integration
- Invoice generation

#### 📧 **Email Service**
- Brevo / Resend providers
- Template support
- Multi-tenant sender configuration

#### 🧠 **Knowledge Base (RAG)**
- Document upload & chunking
- Vector-powered semantic search
- FAQ management
- Product catalog integration

---

## 📁 Architecture

### Directory Structure

```
micro-system-ai-agent-be/
├── cmd/                          # Application entry points
│   ├── saas-api/                 # ✅ Current: Base SaaS module
│   ├── umkm-api/                 # 🔮 Future: UMKM vertical
│   ├── pharmacy-api/             # 🔮 Future: Pharmacy vertical
│   └── manufacturing-api/        # 🔮 Future: Manufacturing vertical
│
├── internal/
│   ├── core/                     # ✅ CORE (Reusable across ALL verticals)
│   │   ├── llm/                  # Multi-LLM provider
│   │   ├── ocr/                  # OCR engine + LLM parsing
│   │   ├── whatsapp/             # WhatsApp integration
│   │   ├── auth/                 # Authentication & JWT
│   │   ├── upload/               # File upload (multi-provider)
│   │   ├── vector/               # Vector DB (Qdrant + embeddings) 🆕
│   │   ├── workflow/             # Workflow automation engine
│   │   ├── kb/                   # Knowledge base / RAG
│   │   ├── email/                # Email service
│   │   ├── payment/              # Payment gateway
│   │   └── notification/         # Notifications
│   │
│   ├── shared/                   # ✅ Shared infrastructure
│   │   ├── config/               # Configuration management
│   │   ├── database/             # Database connection
│   │   ├── errors/               # Error handling
│   │   ├── middleware/           # HTTP middleware
│   │   └── utils/                # Utilities
│   │
│   └── modules/                  # ⚙️ VERTICAL-SPECIFIC modules
│       └── saas/                 # Base SaaS module (COMPLETE)
│           ├── handlers/         # HTTP handlers
│           ├── services/         # Business logic
│           ├── repositories/     # Data access
│           └── models/           # Data models
│
├── migrations/
│   ├── core/                     # Core tables (clients, workflows, etc.)
│   └── saas/                     # SaaS-specific tables
│
└── documentation/                # 20+ documentation files
```

### Architecture Pattern

```
┌─────────────────────────────────────────────────────────────┐
│                    PLATFORM CORE                            │
│  Multi-LLM │ WhatsApp │ OCR │ Auth │ Upload │ Vector DB    │
│  Workflow │ KB/RAG │ Email │ Payment │ Notification         │
└──────────────────┬──────────────────────────────────────────┘
                   │
        ┌──────────┴──────────┬──────────────┬────────────┐
        │                     │              │            │
┌───────▼────────┐  ┌────────▼────────┐  ┌──▼───────┐  ┌▼────────┐
│  SaaS Module   │  │ UMKM Module     │  │ Pharmacy │  │  Future │
│  (Complete)    │  │ (Planned)       │  │ (Planned)│  │ Modules │
└────────────────┘  └─────────────────┘  └──────────┘  └─────────┘
```

---

## 🏁 Quick Start

### Prerequisites

- Go 1.25+
- PostgreSQL 14+
- Docker (optional, for Qdrant)

### 1. Clone Repository

```bash
git clone <repository-url>
cd micro-system-ai-agent-be
```

### 2. Setup Environment

```bash
cp .env.example .env
# Edit .env with your configuration
```

### 3. Install Dependencies

```bash
go mod download
```

### 4. Run Migrations

```bash
# Core migrations
go run cmd/migrate/main.go -dir migrations/core -direction up

# SaaS module migrations
go run cmd/migrate/main.go -dir migrations/saas -direction up
```

### 5. (Optional) Start Qdrant for Vector Search

```bash
docker run -p 6333:6333 -p 6334:6334 qdrant/qdrant
```

### 6. Run Server

```bash
go run cmd/saas-api/main.go
```

Server will start at `http://localhost:8080`

---

## 📚 Documentation

Comprehensive documentation available in [`/documentation`](./documentation/) folder:

### Implementation Guides
- [Authentication System](./documentation/AUTH_IMPLEMENTATION_SUMMARY.md)
- [Product Management](./documentation/PHASE_2_PRODUCT_MANAGEMENT_SUMMARY.md)
- [File Upload System](./documentation/PHASE_3_FILE_UPLOAD_SUMMARY.md)
- [Payment Gateway](./documentation/PAYMENT_IMPLEMENTATION_SUMMARY.md)
- [Workflow Automation](./documentation/WORKFLOW_GUIDE.md)

### API Documentation
- [OCR API](./documentation/OCR_API.md)
- [Knowledge Base API](./documentation/KNOWLEDGE_BASE_API.md)
- [Swagger Docs](./documentation/README_SWAGGER.md)

### Architecture & Planning
- [Product Requirements Document (PRD)](./Product-Requirements-Document.md)
- [Backend Audit Report](./documentation/BACKEND_AUDIT_REPORT.md)
- [Code Review & Improvements](./documentation/CODE_REVIEW_AND_IMPROVEMENTS.md)
- [Template Guide](./TEMPLATE_GUIDE.md) - How to clone for new verticals

---

## 🔧 Configuration

### Environment Variables

Key configuration options (see `.env.example` for complete list):

```bash
# Database
DATABASE_URL=postgresql://user:password@localhost:5432/database

# Server
PORT=8080
ENV=development

# OpenAI (for LLM & Embeddings)
OPENAI_API_KEY=your-api-key

# Authentication
JWT_SECRET=your-jwt-secret
GOOGLE_CLIENT_ID=your-google-client-id

# Upload Provider
UPLOAD_PROVIDER=local  # or cloudinary, s3

# Vector Database
VECTOR_PROVIDER=qdrant_self_hosted  # or qdrant_cloud
QDRANT_HOST=localhost
QDRANT_PORT=6334

# Embedding
EMBEDDING_PROVIDER=openai
EMBEDDING_MODEL=text-embedding-3-small
```

---

## 🧪 Testing

### Run Tests

```bash
# All tests
go test ./...

# Specific package
go test ./internal/core/auth/...

# With coverage
go test -cover ./...
```

### Manual API Testing

```bash
# Register user
curl -X POST http://localhost:8080/auth/register \
  -H "Content-Type: application/json" \
  -d '{"email": "test@example.com", "password": "password123", ...}'

# Login
curl -X POST http://localhost:8080/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email": "test@example.com", "password": "password123"}'

# Access protected route
curl -X GET http://localhost:8080/auth/me \
  -H "Authorization: Bearer <your-access-token>"
```

---

## 🎨 Creating New Vertical Modules

See [TEMPLATE_GUIDE.md](./TEMPLATE_GUIDE.md) for detailed instructions on:

1. Cloning this repository for a new vertical
2. Creating vertical-specific modules
3. Reusing core services
4. Setting up migrations
5. Best practices

**Example: Creating UMKM Module**

```go
// cmd/umkm-api/main.go
import (
    // Reuse CORE services
    "github.com/.../internal/core/llm"
    "github.com/.../internal/core/whatsapp"
    "github.com/.../internal/core/auth"

    // UMKM-specific modules
    "github.com/.../internal/modules/umkm/handlers"
    "github.com/.../internal/modules/umkm/services"
)

func main() {
    // Initialize core services (reusable)
    llmService := llm.NewService(cfg)
    authService := auth.NewService(db, cfg)

    // Initialize UMKM-specific services
    inventoryService := services.NewInventoryService(db)
    posService := services.NewPOSService(db, waService)

    // ... setup routes and run
}
```

---

## 🤝 Contributing

### Development Workflow

1. Create feature branch: `git checkout -b feature/my-feature`
2. Make changes and test
3. Commit: `git commit -m "feat: add my feature"`
4. Push: `git push origin feature/my-feature`
5. Create Pull Request

### Code Style

- Follow Go best practices
- Use `gofmt` for formatting
- Write meaningful commit messages
- Add tests for new features
- Update documentation

### Guidelines

✅ **DO:**
- Keep core services generic and reusable
- Use interfaces for flexibility
- Write tests for critical paths
- Document your code
- Follow the repository pattern

❌ **DON'T:**
- Add industry-specific logic to core
- Create tight coupling between modules
- Skip error handling
- Commit sensitive data (`.env` files)

---

## 📊 Project Status

### ✅ Completed Features

- Multi-LLM support (OpenAI, Gemini, Groq, DeepSeek)
- WhatsApp integration (WAHA)
- OCR with LLM parsing
- Authentication (JWT + OAuth)
- Product management (CRUD)
- File upload (multi-provider)
- Vector database (Qdrant + embeddings) 🆕
- Workflow automation
- Knowledge base (RAG)
- Payment gateway (manual + Midtrans)
- Email service

### 🚧 In Progress

- UMKM vertical module
- Pharmacy vertical module
- Advanced analytics dashboard

### 🔮 Planned

- Manufacturing vertical module
- Voice message support
- Multilingual AI responses
- Mobile app integration

---

## 📞 Support & Contact

- **Documentation**: See `/documentation` folder
- **Issues**: [GitHub Issues](https://github.com/your-repo/issues)
- **Discussions**: [GitHub Discussions](https://github.com/your-repo/discussions)

---

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

---

## 🙏 Acknowledgments

Built with:
- [Fiber](https://gofiber.io/) - Web framework
- [GORM](https://gorm.io/) - ORM
- [Qdrant](https://qdrant.tech/) - Vector database
- [OpenAI](https://openai.com/) - LLM & Embeddings
- [WAHA](https://github.com/devlikeapro/waha) - WhatsApp HTTP API

---

**Version**: 1.0.0
**Last Updated**: January 2026
**Maintained by**: Development Team
