# Architecture Patterns & Best Practices

Deep dive into **Core vs Module** design patterns and when to use each.

---

## 🎯 Core Philosophy

**Core Services** = Reusable building blocks (like LEGO pieces)
**Module Services** = Business logic orchestration (how you use the LEGOs)

**Golden Rule**: If it's **industry-agnostic**, it belongs in **Core**. If it's **business-specific**, it belongs in **Module**.

---

## 🔧 Pattern 1: Core Provides Tools, Modules Provide Logic

### Example: Analytics & Reporting

#### ❌ **WRONG** - Full implementation in Core
```go
// DON'T DO THIS - Too specific!
// internal/core/analytics/sales_analytics.go

func GetDrugSalesTrend() {
    // ❌ This is Pharmacy-specific, NOT generic
    db.Query("SELECT drug_category, SUM(amount) FROM pharmacy_sales...")
}
```

#### ✅ **RIGHT** - Generic helpers in Core, specific logic in Module

**Core (Reusable Helpers):**
```go
// internal/core/analytics/aggregator.go

type AggregateQuery struct {
    Table      string
    GroupBy    []string
    Aggregates map[string]string // {"total": "SUM(amount)", "count": "COUNT(*)"}
    Filters    map[string]interface{}
    DateRange  *DateRange
    OrderBy    []string
}

// Generic aggregation function
func (a *Aggregator) Aggregate(query AggregateQuery) ([]map[string]interface{}, error) {
    // Build SQL dynamically based on query
    sql := a.buildSQL(query)
    return db.Query(sql, query.Filters)
}

// Generic chart data formatter
func ToLineChartData(data []map[string]interface{}, xKey, yKey string) ChartData {
    // Convert raw query results to chart format
}

func ToPieChartData(data []map[string]interface{}, labelKey, valueKey string) ChartData {
    // Convert to pie chart format
}
```

**Module (Business Logic):**
```go
// internal/modules/farmasi/services/analytics_service.go

type FarmasiAnalyticsService struct {
    aggregator *analytics.Aggregator // Use core helper
    repo       *FarmasiSalesRepository
}

func (s *FarmasiAnalyticsService) GetDrugSalesTrend(clientID uuid.UUID, startDate, endDate time.Time) ChartData {
    // 1. Use core aggregator with Farmasi-specific query
    results, _ := s.aggregator.Aggregate(analytics.AggregateQuery{
        Table: "farmasi_sales JOIN farmasi_drugs ON farmasi_sales.drug_id = farmasi_drugs.id",
        GroupBy: []string{"DATE(sold_at)", "drug_category"},
        Aggregates: map[string]string{
            "total_amount": "SUM(amount)",
            "total_qty": "SUM(quantity)",
        },
        Filters: map[string]interface{}{
            "client_id": clientID,
            "sold_at BETWEEN ? AND ?": []interface{}{startDate, endDate},
        },
        OrderBy: []string{"DATE(sold_at) ASC"},
    })

    // 2. Use core formatter to convert to chart data
    return analytics.ToLineChartData(results, "sold_at", "total_amount")
}

func (s *FarmasiAnalyticsService) GetTopSellingDrugs(clientID uuid.UUID, limit int) ChartData {
    results, _ := s.aggregator.Aggregate(analytics.AggregateQuery{
        Table: "farmasi_sales JOIN farmasi_drugs ON farmasi_sales.drug_id = farmasi_drugs.id",
        GroupBy: []string{"drug_name"},
        Aggregates: map[string]string{
            "total_sold": "SUM(quantity)",
        },
        Filters: map[string]interface{}{
            "client_id": clientID,
        },
        OrderBy: []string{"total_sold DESC"},
    })

    return analytics.ToPieChartData(results, "drug_name", "total_sold")
}
```

**Benefits:**
- ✅ Core helpers reusable for UMKM, Manufacturing, etc.
- ✅ Each module has full control over their ERD & business logic
- ✅ No duplication of chart formatting code

---

## 📊 Pattern 2: Audit Log Service

### Core Implementation (Generic Tracking)

```go
// internal/core/audit/service.go

type AuditService struct {
    db *gorm.DB
}

type AuditLog struct {
    ID        uuid.UUID
    UserID    uuid.UUID
    ClientID  uuid.UUID
    Action    string // "create", "update", "delete", "view"
    Entity    string // "product", "drug", "order"
    EntityID  string
    OldValue  JSONB  // Previous state
    NewValue  JSONB  // New state
    IPAddress string
    UserAgent string
    CreatedAt time.Time
}

// Generic audit logging
func (s *AuditService) Log(ctx context.Context, log *AuditLog) error {
    return s.db.Create(log).Error
}

func (s *AuditService) GetLogs(filters AuditFilter) ([]AuditLog, error) {
    // Generic query with filters
}
```

### Module Usage (Business Context)

```go
// internal/modules/farmasi/services/drug_service.go

func (s *DrugService) UpdateDrug(ctx context.Context, drugID uuid.UUID, updates *UpdateDrugRequest) error {
    // 1. Get old value
    oldDrug, _ := s.repo.GetByID(drugID)

    // 2. Update drug
    newDrug, _ := s.repo.Update(drugID, updates)

    // 3. Log audit trail using core service
    s.auditService.Log(ctx, &audit.AuditLog{
        UserID:   getUserID(ctx),
        ClientID: getClientID(ctx),
        Action:   "update",
        Entity:   "drug",
        EntityID: drugID.String(),
        OldValue: toJSON(oldDrug),
        NewValue: toJSON(newDrug),
    })

    return nil
}
```

**Why This Works:**
- ✅ Core audit service is industry-agnostic
- ✅ Modules add business context (what entity, what changed)
- ✅ Compliance reports can query audit logs across all modules

---

## 📤 Pattern 3: File Export Service (PDF/Excel)

### Core Implementation (Generic Exporters)

```go
// internal/core/export/pdf.go

type PDFExporter struct {
    pdf *gopdf.GoPdf
}

// Generic table rendering
func (e *PDFExporter) RenderTable(headers []string, rows [][]string) error {
    // Draw table with headers and rows
}

// Generic invoice template
func (e *PDFExporter) RenderInvoice(invoice InvoiceData) ([]byte, error) {
    // Generic invoice layout
    e.AddTitle(invoice.Title)
    e.AddText("Invoice #: " + invoice.Number)
    e.AddText("Date: " + invoice.Date.Format("2006-01-02"))
    e.RenderTable([]string{"Item", "Qty", "Price", "Total"}, invoice.Items)
    e.AddText("Total: " + fmt.Sprintf("%.2f", invoice.Total))
    return e.GetBytes()
}
```

```go
// internal/core/export/excel.go

type ExcelExporter struct {
    file *excelize.File
}

// Generic Excel export from query results
func (e *ExcelExporter) FromQueryResults(headers []string, data []map[string]interface{}) ([]byte, error) {
    // Create Excel with headers and data
}

// Generic styling
func (e *ExcelExporter) ApplyHeaderStyle() {
    // Bold, background color, etc.
}
```

### Module Usage (Business Data)

```go
// internal/modules/farmasi/services/export_service.go

type FarmasiExportService struct {
    pdfExporter   *export.PDFExporter
    excelExporter *export.ExcelExporter
    salesRepo     *FarmasiSalesRepository
}

// Export sales report as PDF
func (s *FarmasiExportService) ExportSalesReportPDF(clientID uuid.UUID, month time.Time) ([]byte, error) {
    // 1. Get sales data from Farmasi ERD
    sales, _ := s.salesRepo.GetSalesByMonth(clientID, month)

    // 2. Convert to generic invoice format
    invoiceData := export.InvoiceData{
        Title:  "Monthly Sales Report - " + month.Format("January 2006"),
        Number: fmt.Sprintf("RPT-%s-%s", clientID.String()[:8], month.Format("200601")),
        Date:   month,
        Items:  s.convertSalesToItems(sales), // Convert Farmasi sales to generic items
        Total:  s.calculateTotal(sales),
    }

    // 3. Use core PDF exporter
    return s.pdfExporter.RenderInvoice(invoiceData)
}

// Export drug inventory as Excel
func (s *FarmasiExportService) ExportDrugInventoryExcel(clientID uuid.UUID) ([]byte, error) {
    // 1. Query Farmasi ERD
    drugs, _ := s.drugRepo.GetAll(clientID)

    // 2. Convert to generic format
    headers := []string{"Drug Name", "Category", "Stock", "Price", "Expiry Date"}
    data := make([]map[string]interface{}, len(drugs))
    for i, drug := range drugs {
        data[i] = map[string]interface{}{
            "Drug Name":  drug.Name,
            "Category":   drug.Category,
            "Stock":      drug.Stock,
            "Price":      drug.Price,
            "Expiry Date": drug.ExpiryDate.Format("2006-01-02"),
        }
    }

    // 3. Use core Excel exporter
    return s.excelExporter.FromQueryResults(headers, data)
}
```

**Pattern Benefits:**
- ✅ Core provides PDF/Excel rendering engine
- ✅ Modules provide business-specific data mapping
- ✅ Invoice template reusable across all verticals
- ✅ Each module can export their own ERD structure

---

## 🔄 Pattern 4: Background Jobs / Queue

### Core Implementation (Job Infrastructure)

```go
// internal/core/queue/queue.go

type JobQueue struct {
    redis  *redis.Client
    workers int
}

type Job struct {
    ID      string
    Type    string // "send_email", "generate_report", "import_data"
    Payload map[string]interface{}
    Status  string // "pending", "processing", "completed", "failed"
    Retries int
    MaxRetries int
}

// Generic job enqueue
func (q *JobQueue) Enqueue(job *Job) error {
    return q.redis.RPush("queue:jobs", job).Err()
}

// Generic job processor
func (q *JobQueue) Process(handler JobHandler) {
    for {
        job := q.Dequeue()
        handler.Handle(job)
    }
}

type JobHandler interface {
    Handle(job *Job) error
}
```

### Module Usage (Business Jobs)

```go
// internal/modules/farmasi/jobs/handlers.go

type FarmasiJobHandler struct {
    exportService *FarmasiExportService
    emailService  *email.Service
}

func (h *FarmasiJobHandler) Handle(job *queue.Job) error {
    switch job.Type {
    case "generate_monthly_report":
        clientID := uuid.MustParse(job.Payload["client_id"].(string))
        month := parseDate(job.Payload["month"].(string))

        // Generate report using Farmasi export service
        pdfBytes, err := h.exportService.ExportSalesReportPDF(clientID, month)
        if err != nil {
            return err
        }

        // Send via email
        return h.emailService.SendWithAttachment(
            job.Payload["email"].(string),
            "Monthly Sales Report",
            "Please find attached your monthly sales report.",
            pdfBytes,
            "sales-report.pdf",
        )

    case "import_drug_catalog":
        // Handle bulk drug import
        return h.importDrugsFromCSV(job.Payload["file_path"].(string))
    }

    return nil
}
```

**Why This Pattern Works:**
- ✅ Core provides job queue infrastructure (Redis, workers, retry logic)
- ✅ Modules define their own job types & handlers
- ✅ Easy to add new job types per vertical

---

## 🎨 Pattern 5: SaaS Module - Core Service Orchestration

### Analysis: What's in SaaS Module?

**Files using CORE services (Wrappers/Orchestrators):**

| File | Core Services Used | Purpose |
|------|-------------------|---------|
| `ocr_handler.go` | `core/ocr` | Expose OCR API endpoint |
| `whatsapp_handler.go` | `core/whatsapp` | WhatsApp session management API |
| `kb_handler.go` | `core/kb` + `core/vector` | Knowledge base API |
| `payment_handler.go` | `core/payment` | Payment processing API |
| `workflow_handler.go` | `core/workflow` | Workflow automation API |
| `webhook_handler.go` | `core/llm` + `core/whatsapp` + `core/kb` + `core/ocr` | **Multi-service orchestration** ⭐ |

**Files with UNIQUE business logic (SaaS-specific):**

| File | Purpose | Why Unique? |
|------|---------|-------------|
| `cart_*.go` | Shopping cart | E-commerce flow |
| `product_*.go` | Product catalog | Inventory management |
| `order_*.go` | Order processing | Transaction lifecycle |
| `client_*.go` | Tenant management | Multi-tenancy configuration |
| `conversation_*.go` | Chat history | WhatsApp conversation tracking |
| `transaction_*.go` | Transaction logs | Business transaction records (from OCR) |
| `credit_*.go` | Credit/balance | Payment & subscription credits |
| `message_service.go` | Business messaging | WhatsApp message templates & logic |

### Example: Webhook Service (Best Practice Orchestration)

```go
// internal/modules/saas/services/webhook_service.go

type WebhookService struct {
    // Core services (injected)
    llmService       *llm.Service
    whatsappService  *whatsapp.Service
    kbRetriever      *kb.VectorRetriever
    ocrService       *ocr.Service

    // SaaS-specific repositories
    conversationRepo *ConversationRepository
    transactionRepo  *TransactionRepository
    clientRepo       *ClientRepository
}

func (s *WebhookService) ProcessIncomingMessage(clientID uuid.UUID, message WhatsAppMessage) error {
    // 1. Get client context
    client, _ := s.clientRepo.GetByID(clientID)

    // 2. If message has image, use CORE OCR
    if message.HasImage {
        ocrResult, _ := s.ocrService.ProcessImage(message.ImageURL)

        // Save transaction (SaaS-specific)
        s.transactionRepo.Create(&Transaction{
            ClientID: clientID,
            Amount:   ocrResult.TotalAmount,
            Items:    ocrResult.Items,
            RawOCR:   ocrResult.RawText,
        })

        return s.whatsappService.SendMessage(message.From, "Receipt processed!")
    }

    // 3. Get relevant context from CORE KB
    context, _ := s.kbRetriever.GetRelevantContext(clientID.String(), message.Text, 5)

    // 4. Generate response using CORE LLM
    prompt := fmt.Sprintf("Context:\n%s\n\nUser: %s\nAssistant:", context, message.Text)
    response, _ := s.llmService.Generate(prompt)

    // 5. Save conversation (SaaS-specific)
    s.conversationRepo.Create(&Conversation{
        ClientID:     clientID,
        UserMessage:  message.Text,
        BotResponse:  response,
        KBContext:    context,
    })

    // 6. Send response via CORE WhatsApp
    return s.whatsappService.SendMessage(message.From, response)
}
```

**Key Insight**:
- ✅ Uses **5 core services** (LLM, WhatsApp, KB, OCR, Vector)
- ✅ Adds **SaaS business logic** (save conversations, transactions)
- ✅ **Orchestrates** the full flow

---

## 📋 Decision Matrix: Core vs Module?

Use this checklist when deciding where to put code:

### Put in **CORE** if:
- ✅ Industry-agnostic (works for Pharmacy, UMKM, Manufacturing)
- ✅ No business logic, just technical infrastructure
- ✅ Reusable helper/utility function
- ✅ External service integration (OpenAI, Qdrant, Cloudinary)
- ✅ Generic data transformation (JSON to PDF, CSV to Excel)

**Examples:**
- LLM provider interface ✅
- Vector database operations ✅
- Chart data formatting helpers ✅
- PDF/Excel rendering engines ✅
- Job queue infrastructure ✅

### Put in **MODULE** if:
- ✅ Depends on module-specific ERD (database tables)
- ✅ Contains business rules (pricing, validation, workflows)
- ✅ Industry-specific logic (drug interactions, stock alerts)
- ✅ Orchestrates multiple core services for a business flow
- ✅ Domain-specific data models

**Examples:**
- Drug sales analytics (Farmasi ERD) ✅
- Shopping cart logic (E-commerce flow) ✅
- Prescription validation (Pharmacy rules) ✅
- Production scheduling (Manufacturing) ✅
- WhatsApp conversation orchestration ✅

---

## 🚀 Future Core Services (Planned)

Based on analysis, these would be good **Core** additions:

### 1. **Analytics Helpers** (`internal/core/analytics/`)
```go
- AggregateQuery builder
- Chart data formatters (line, bar, pie)
- Date range utilities
- Export formatters (CSV, Excel, JSON)
```

**Reusability**: All modules need analytics!

### 2. **Export Engine** (`internal/core/export/`)
```go
- PDF renderer (invoices, reports)
- Excel generator (from query results)
- QR code / Barcode generator
- CSV export helper
```

**Reusability**: All modules generate reports!

### 3. **Job Queue** (`internal/core/queue/`)
```go
- Redis-based job queue
- Worker pool management
- Retry mechanism
- Job scheduling
```

**Reusability**: Background tasks for all modules!

### 4. **Audit Logger** (`internal/core/audit/`)
```go
- Generic audit log storage
- Query/filter audit logs
- Compliance report generation
```

**Reusability**: Regulatory compliance (Farmasi, Manufacturing need this!)

---

## ✅ Best Practices Summary

1. **Core = Tools, Module = Business Logic**
   - Core provides building blocks
   - Modules orchestrate and add business context

2. **Avoid Over-Engineering Core**
   - Don't add to core until you see the pattern in 2+ modules
   - Better to duplicate initially, then extract common pattern

3. **Dependency Direction**
   - ✅ Module → Core (allowed)
   - ❌ Core → Module (forbidden!)

4. **Testing Strategy**
   - Core: Unit tests (no business context)
   - Module: Integration tests (with business scenarios)

5. **When in Doubt**
   - Start in Module (specific)
   - Refactor to Core when you see reusability (generic)

---

**This pattern makes the platform incredibly powerful!** Each new vertical gets:
- 70%+ code reuse from Core
- 30% custom business logic
- 1-2 weeks to production instead of months 🚀

---

**Last Updated**: January 16, 2026
**Reference**: See `TEMPLATE_GUIDE.md` for implementation examples
