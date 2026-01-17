# Template Guide: Cloning for New Verticals

This guide explains how to use `micro-system-ai-agent-be` as a **base template** for creating new vertical-specific projects (e.g., Farmasi, UMKM, FNB).

---

## 🎯 Overview

The platform is designed as a **reusable foundation**. When creating a new vertical (e.g., Pharmacy), you:

1. **Clone** this repository
2. **Rename** the project
3. **Keep** all core services (`internal/core/`)
4. **Replace** `internal/modules/saas/` with your vertical module
5. **Add** vertical-specific migrations
6. **Customize** for your industry

---

## 📋 Step-by-Step Guide

### Step 1: Clone the Repository

```bash
# Clone the base template
git clone https://github.com/your-org/micro-system-ai-agent-be.git micro-system-farmasi-be

# Enter directory
cd micro-system-farmasi-be

# Remove old git history (optional)
rm -rf .git
git init
git add .
git commit -m "Initial commit from template"
```

---

### Step 2: Rename Go Module

Update `go.mod`:

```diff
- module github.com/MuhamadAgungGumelar/micro-system-ai-agent-be
+ module github.com/YourOrg/micro-system-farmasi-be
```

Find and replace all imports:

```bash
# macOS/Linux
find . -type f -name "*.go" -exec sed -i '' 's|github.com/MuhamadAgungGumelar/micro-system-ai-agent-be|github.com/YourOrg/micro-system-farmasi-be|g' {} +

# Windows PowerShell
Get-ChildItem -Recurse -Filter *.go | ForEach-Object {
    (Get-Content $_.FullName) -replace 'github.com/MuhamadAgungGumelar/micro-system-ai-agent-be', 'github.com/YourOrg/micro-system-farmasi-be' |
    Set-Content $_.FullName
}
```

---

### Step 3: Rename Vertical Module

**Option A: Rename SaaS Module** (if your vertical is similar)

```bash
# Rename folder
mv internal/modules/saas internal/modules/farmasi

# Rename migrations folder
mv migrations/saas migrations/farmasi
```

Then update all imports and table names.

**Option B: Create New Module from Scratch** (recommended)

```bash
# Remove SaaS module
rm -rf internal/modules/saas
rm -rf migrations/saas

# Create new vertical module
mkdir -p internal/modules/farmasi/{handlers,services,repositories,models}
mkdir -p migrations/farmasi
```

---

### Step 4: Create Entry Point

Create `cmd/farmasi-api/main.go`:

```go
package main

import (
	"log"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"

	// ✅ Import CORE services (reusable)
	"github.com/YourOrg/micro-system-farmasi-be/internal/core/auth"
	"github.com/YourOrg/micro-system-farmasi-be/internal/core/llm"
	"github.com/YourOrg/micro-system-farmasi-be/internal/core/ocr"
	"github.com/YourOrg/micro-system-farmasi-be/internal/core/upload"
	"github.com/YourOrg/micro-system-farmasi-be/internal/core/vector"
	"github.com/YourOrg/micro-system-farmasi-be/internal/core/whatsapp"
	"github.com/YourOrg/micro-system-farmasi-be/internal/core/workflow"

	// ✅ Import SHARED infrastructure
	"github.com/YourOrg/micro-system-farmasi-be/internal/shared/config"
	"github.com/YourOrg/micro-system-farmasi-be/internal/shared/database"

	// ✅ Import FARMASI-specific modules
	farmasiHandlers "github.com/YourOrg/micro-system-farmasi-be/internal/modules/farmasi/handlers"
	farmasiServices "github.com/YourOrg/micro-system-farmasi-be/internal/modules/farmasi/services"
	farmasiRepos "github.com/YourOrg/micro-system-farmasi-be/internal/modules/farmasi/repositories"
)

func main() {
	// Load configuration
	cfg := config.LoadConfig()

	// Connect to database
	db, err := database.Connect(cfg.DatabaseURL)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	// ========================================
	// Initialize CORE Services (REUSABLE!)
	// ========================================

	// LLM Service
	llmService, err := llm.NewService(cfg)
	if err != nil {
		log.Fatal("Failed to initialize LLM:", err)
	}

	// WhatsApp Service
	waService := whatsapp.NewService(cfg.WameoAPIURL, cfg.WameoAPIKey)

	// OCR Service
	ocrService, err := ocr.NewService(cfg, llmService)
	if err != nil {
		log.Fatal("Failed to initialize OCR:", err)
	}

	// Auth Service
	authService := auth.NewService(db, cfg)

	// Upload Service (choose provider)
	var uploadProvider upload.Provider
	switch cfg.UploadProvider {
	case "cloudinary":
		uploadProvider, _ = upload.NewCloudinaryProvider(
			cfg.CloudinaryCloudName,
			cfg.CloudinaryAPIKey,
			cfg.CloudinaryAPISecret,
		)
	case "s3":
		uploadProvider, _ = upload.NewS3Provider(
			cfg.S3AccessKeyID,
			cfg.S3SecretAccessKey,
			cfg.S3Region,
			cfg.S3BucketName,
		)
	default:
		uploadProvider, err = upload.NewLocalProvider(cfg.UploadBasePath, cfg.UploadBaseURL)
		if err != nil {
			log.Fatalf("Failed to initialize local storage: %v", err)
		}
	}
	uploadService := upload.NewService(uploadProvider)

	// Vector Database Service
	var vectorProvider vector.Provider
	var embeddingProvider vector.EmbeddingProvider

	// Initialize embedding provider
	embeddingProvider, _ = vector.NewOpenAIEmbeddingProvider(cfg.OpenAIKey, cfg.EmbeddingModel)

	// Initialize vector provider
	switch cfg.VectorProvider {
	case "qdrant_cloud":
		vectorProvider, _ = vector.NewQdrantCloudProvider(cfg.QdrantCloudURL, cfg.QdrantCloudAPIKey)
	default:
		vectorProvider, _ = vector.NewQdrantSelfHostedProvider(cfg.QdrantSelfHostedHost, cfg.QdrantSelfHostedPort)
	}

	vectorService := vector.NewService(vectorProvider, embeddingProvider)

	// Workflow Service
	workflowScheduler := workflow.NewScheduler()
	workflowScheduler.Start()

	// ========================================
	// Initialize FARMASI-Specific Services
	// ========================================

	// Repositories
	drugRepo := farmasiRepos.NewDrugRepository(db)
	prescriptionRepo := farmasiRepos.NewPrescriptionRepository(db)
	patientRepo := farmasiRepos.NewPatientRepository(db)

	// Services
	drugService := farmasiServices.NewDrugService(drugRepo)
	prescriptionService := farmasiServices.NewPrescriptionService(prescriptionRepo, ocrService)
	patientService := farmasiServices.NewPatientService(patientRepo, waService)

	// Handlers
	drugHandler := farmasiHandlers.NewDrugHandler(drugService)
	prescriptionHandler := farmasiHandlers.NewPrescriptionHandler(prescriptionService)
	patientHandler := farmasiHandlers.NewPatientHandler(patientService)

	// ========================================
	// Setup HTTP Server
	// ========================================

	app := fiber.New(fiber.Config{
		AppName: "Farmasi API v1.0",
	})

	// Middleware
	app.Use(logger.New())
	app.Use(cors.New())

	// Health check
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "healthy"})
	})

	// ========================================
	// API Routes
	// ========================================

	api := app.Group("/api/v1")

	// Auth routes (from CORE)
	authGroup := api.Group("/auth")
	authHandler := auth.NewHandler(authService)
	authGroup.Post("/register", authHandler.Register)
	authGroup.Post("/login", authHandler.Login)
	authGroup.Post("/refresh", authHandler.RefreshToken)
	authGroup.Post("/logout", authHandler.Logout)
	authGroup.Get("/me", auth.AuthMiddleware(authService), authHandler.GetCurrentUser)

	// Farmasi-specific routes (PROTECTED)
	farmasiGroup := api.Group("/farmasi", auth.AuthMiddleware(authService))

	// Drugs
	farmasiGroup.Post("/drugs", drugHandler.CreateDrug)
	farmasiGroup.Get("/drugs", drugHandler.ListDrugs)
	farmasiGroup.Get("/drugs/:id", drugHandler.GetDrug)
	farmasiGroup.Put("/drugs/:id", drugHandler.UpdateDrug)
	farmasiGroup.Delete("/drugs/:id", drugHandler.DeleteDrug)

	// Prescriptions
	farmasiGroup.Post("/prescriptions", prescriptionHandler.CreatePrescription)
	farmasiGroup.Post("/prescriptions/scan", prescriptionHandler.ScanPrescription) // OCR
	farmasiGroup.Get("/prescriptions", prescriptionHandler.ListPrescriptions)
	farmasiGroup.Get("/prescriptions/:id", prescriptionHandler.GetPrescription)

	// Patients
	farmasiGroup.Post("/patients", patientHandler.CreatePatient)
	farmasiGroup.Get("/patients", patientHandler.ListPatients)
	farmasiGroup.Post("/patients/:id/reminder", patientHandler.SendReminder) // WhatsApp

	// ========================================
	// Start Server
	// ========================================

	log.Printf("✅ Farmasi API running at :%s", cfg.Port)
	log.Fatal(app.Listen(":" + cfg.Port))
}
```

---

### Step 5: Create Vertical-Specific Models

Example: `internal/modules/farmasi/models/drug.go`

```go
package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Drug represents a pharmaceutical drug in the database
type Drug struct {
	ID          uuid.UUID `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	ClientID    uuid.UUID `json:"client_id" gorm:"type:uuid;not null;index"` // Multi-tenant

	// Drug information
	Name        string  `json:"name" gorm:"not null"`
	GenericName string  `json:"generic_name"`
	Category    string  `json:"category"` // "Antibiotic", "Painkiller", etc.
	Dosage      string  `json:"dosage"`   // "500mg", "10ml", etc.
	Form        string  `json:"form"`     // "Tablet", "Syrup", "Injection"

	// Stock & pricing
	Stock       int     `json:"stock" gorm:"default:0"`
	Price       float64 `json:"price" gorm:"type:decimal(12,2);default:0"`
	SKU         string  `json:"sku" gorm:"index"`

	// Regulatory
	BPOMNumber  string `json:"bpom_number"`
	IsControlled bool  `json:"is_controlled"` // Controlled substance?
	ExpiryDate  *time.Time `json:"expiry_date"`

	// Status
	IsActive    bool          `json:"is_active" gorm:"default:true"`
	CreatedAt   time.Time     `json:"created_at"`
	UpdatedAt   time.Time     `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `json:"-" gorm:"index"` // Soft delete
}

func (Drug) TableName() string {
	return "farmasi_drugs"
}
```

---

### Step 6: Create Migrations

Create `migrations/farmasi/000001_create_drugs.up.sql`:

```sql
-- Drugs table
CREATE TABLE IF NOT EXISTS farmasi_drugs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    client_id UUID NOT NULL REFERENCES clients(id) ON DELETE CASCADE,

    name TEXT NOT NULL,
    generic_name TEXT,
    category TEXT,
    dosage TEXT,
    form TEXT,

    stock INTEGER NOT NULL DEFAULT 0,
    price DECIMAL(12,2) NOT NULL DEFAULT 0,
    sku TEXT,

    bpom_number TEXT,
    is_controlled BOOLEAN DEFAULT false,
    expiry_date TIMESTAMP,

    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    deleted_at TIMESTAMP
);

-- Indexes
CREATE INDEX idx_farmasi_drugs_client_id ON farmasi_drugs(client_id);
CREATE INDEX idx_farmasi_drugs_category ON farmasi_drugs(category);
CREATE INDEX idx_farmasi_drugs_sku ON farmasi_drugs(sku);
CREATE INDEX idx_farmasi_drugs_expiry_date ON farmasi_drugs(expiry_date);
CREATE INDEX idx_farmasi_drugs_deleted_at ON farmasi_drugs(deleted_at);

-- Unique constraint
CREATE UNIQUE INDEX idx_farmasi_drugs_client_sku
ON farmasi_drugs(client_id, sku)
WHERE sku IS NOT NULL AND deleted_at IS NULL;

-- Comments
COMMENT ON TABLE farmasi_drugs IS 'Pharmaceutical drugs catalog for pharmacy vertical';
COMMENT ON COLUMN farmasi_drugs.is_controlled IS 'True for controlled substances (narcotics, psychotropics)';
```

Down migration (`000001_create_drugs.down.sql`):

```sql
DROP TABLE IF EXISTS farmasi_drugs CASCADE;
```

---

### Step 7: Update Configuration

Update `.env` for your vertical:

```bash
# App Name
APP_NAME=Farmasi API

# Database (vertical-specific database optional)
DATABASE_URL=postgresql://user:password@localhost:5432/farmasi_db

# All other configs remain the same (reusing core services)
OPENAI_API_KEY=...
JWT_SECRET=...
UPLOAD_PROVIDER=cloudinary
VECTOR_PROVIDER=qdrant_cloud
# ...
```

---

## 🎨 Customization Examples

### Example 1: UMKM (Small Business) Vertical

**Modules to create:**
- `internal/modules/umkm/models/` - Inventory, Sales, Customers
- `internal/modules/umkm/services/` - POS service, Accounting service
- `internal/modules/umkm/handlers/` - Inventory, Sales, Reports

**Migrations:**
- `migrations/umkm/000001_create_inventory.up.sql`
- `migrations/umkm/000002_create_sales.up.sql`
- `migrations/umkm/000003_create_customers.up.sql`

**Entry point:**
- `cmd/umkm-api/main.go`

**Reused core services:**
- WhatsApp (for customer orders)
- OCR (for receipt scanning)
- LLM (for customer service chatbot)
- Workflow (for low stock alerts, daily reports)
- Vector DB (for product recommendations)

---

### Example 2: FNB (Food & Beverage) Vertical

**Modules to create:**
- `internal/modules/fnb/models/` - Menu, Orders, Tables
- `internal/modules/fnb/services/` - Order service, Kitchen service
- `internal/modules/fnb/handlers/` - Menu, Orders, Tables

**Migrations:**
- `migrations/fnb/000001_create_menu.up.sql`
- `migrations/fnb/000002_create_orders.up.sql`
- `migrations/fnb/000003_create_tables.up.sql`

**Entry point:**
- `cmd/fnb-api/main.go`

**Reused core services:**
- WhatsApp (for order taking)
- Payment (for billing)
- Workflow (for kitchen automation, order ready notifications)

---

## 📦 What to Keep vs. Replace

### ✅ KEEP (Always Reuse)

**Folders:**
- `internal/core/*` - ALL core services
- `internal/shared/*` - ALL shared infrastructure
- `migrations/core/*` - Core tables (clients, users, etc.)

**Files:**
- `go.mod` (update module name only)
- `.env.example` (update as needed)
- `cmd/migrate/` - Migration tool

### 🔄 REPLACE (Vertical-Specific)

**Folders:**
- `internal/modules/saas/` → `internal/modules/farmasi/`
- `migrations/saas/` → `migrations/farmasi/`
- `cmd/saas-api/` → `cmd/farmasi-api/`

**Files:**
- `Product-Requirements-Document.md` (create vertical-specific PRD)
- `README.md` (update for your vertical)

### ❌ REMOVE (Optional)

**Folders:**
- `/documentation` - Remove or keep as reference

---

## 🚀 Best Practices

### DO ✅

1. **Reuse core services** - Don't duplicate what's in `internal/core/`
2. **Follow the same patterns** - Use Repository → Service → Handler
3. **Multi-tenant everything** - Always include `client_id`
4. **Use soft delete** - Add `deleted_at` to models
5. **Document your code** - Add comments and docs
6. **Write tests** - Especially for business logic
7. **Keep modules independent** - Don't reference other vertical modules

### DON'T ❌

1. **Don't modify core/** - Keep core generic and reusable
2. **Don't hardcode** - Use environment variables
3. **Don't skip migrations** - Always create proper migrations
4. **Don't tight couple** - Modules should be independent
5. **Don't commit secrets** - Use `.env` (not `.env.example`)

---

## 🧪 Testing Your Vertical

After setup, test the integration:

```bash
# Run migrations
go run cmd/migrate/main.go -dir migrations/core -direction up
go run cmd/migrate/main.go -dir migrations/farmasi -direction up

# Run server
go run cmd/farmasi-api/main.go

# Test auth (core service)
curl -X POST http://localhost:8080/api/v1/auth/register \
  -d '{"email":"test@farmasi.com","password":"test123",...}'

# Test vertical-specific endpoint
curl -X GET http://localhost:8080/api/v1/farmasi/drugs \
  -H "Authorization: Bearer <token>"
```

---

## 📞 Support

If you encounter issues:

1. Check [README.md](./README.md) for platform overview
2. Read [Product-Requirements-Document.md](./Product-Requirements-Document.md)
3. Review [documentation/](./documentation/) for implementation guides
4. Open an issue on GitHub

---

## 🎓 Summary Checklist

When cloning for new vertical:

- [ ] Clone repository
- [ ] Rename Go module (`go.mod`)
- [ ] Update all imports
- [ ] Rename/create vertical module folder
- [ ] Create entry point (`cmd/<vertical>-api/main.go`)
- [ ] Reuse core services (LLM, WhatsApp, Auth, etc.)
- [ ] Create vertical-specific models
- [ ] Create migrations (`migrations/<vertical>/`)
- [ ] Update `.env` configuration
- [ ] Test migrations
- [ ] Test API endpoints
- [ ] Update documentation

---

**Happy Building! 🚀**

The platform is designed to make vertical creation fast. You should be able to launch a new vertical in **1-2 weeks** by reusing 70%+ of the platform code.
