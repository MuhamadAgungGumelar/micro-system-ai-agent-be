# CMS Frontend Implementation Guide
## WhatsApp Bot SaaS - Multi-Module Platform

---

## 📋 Table of Contents
1. [System Overview](#system-overview)
2. [Architecture Decision: Monorepo Multi-CMS](#architecture-decision-monorepo-multi-cms)
3. [Frontend Structure](#frontend-structure)
4. [Backend API Base URL](#backend-api-base-url)
5. [Features & Modules](#features--modules)
6. [CMS Pages to Implement](#cms-pages-to-implement)
7. [API Endpoints Reference](#api-endpoints-reference)
8. [Database Schema](#database-schema)
9. [User Roles & Permissions](#user-roles--permissions)
10. [Implementation Priority](#implementation-priority)

---

## 🎯 System Overview

**Product:** Multi-tenant WhatsApp Bot SaaS Platform with Multiple Industry Modules
**Architecture:** Go Backend (Fiber) + Monorepo Multi-CMS Frontend (React/Next.js)
**Integration:** WAHA (WhatsApp HTTP API), OpenAI, Payment Gateway (Manual/Midtrans)

### Business Model:
- **Subscription per Module:** Tenant subscribe to ONE specific module only
- **Module Isolation:** UMKM tenant CANNOT access Farmasi features
- **Independent CMSs:** Each module has its own CMS application
- **Shared Core:** Common UI components and authentication shared across modules

### Modules:
1. **SaaS (E-Commerce)** - Online store order management via WhatsApp
2. **UMKM** - Small business inventory, cashflow, supplier management
3. **Farmasi** - Pharmacy prescription, drug inventory, patient records
4. **Manufaktur** - Production orders, BOM, quality control

### Core Capabilities (All Modules):
- Multi-tenant WhatsApp bot management
- AI-powered customer service via WhatsApp
- Knowledge base management
- Workflow automation
- Admin command system via WhatsApp
- OCR receipt processing

---

## 🏗️ Architecture Decision: Monorepo Multi-CMS

### Why Multiple CMSs Instead of One?

**Business Requirements:**
- ✅ Tenant pays per module → Only access their subscribed module
- ✅ Module-specific features are very different (e.g., Prescriptions vs Production Orders)
- ✅ Bundle size optimization → Tenant only downloads code for their module
- ✅ Independent deployment → Deploy Farmasi updates without affecting SaaS
- ✅ Team scaling → Different teams can work on different modules

**Architecture Pattern: Monorepo with Shared Core**

```
whatsapp-bot-saas/
├── packages/
│   ├── core/                    # ← SHARED across all CMSs
│   │   ├── components/          # Button, Table, Modal, Form, Chart
│   │   ├── layouts/             # DashboardLayout, Sidebar, Header
│   │   ├── hooks/               # useAuth, useAPI, useToast
│   │   ├── services/            # API client, auth service
│   │   ├── utils/               # Formatters, validators, constants
│   │   └── types/               # Shared TypeScript types
│   │
│   ├── cms-saas/                # ← E-Commerce CMS (Independent App)
│   │   ├── src/
│   │   │   ├── pages/
│   │   │   │   ├── Orders/
│   │   │   │   ├── Products/
│   │   │   │   ├── Customers/
│   │   │   │   └── Analytics/
│   │   │   └── App.tsx
│   │   ├── package.json
│   │   └── vercel.json          # Deploy to cms-saas.yourdomain.com
│   │
│   ├── cms-umkm/                # ← UMKM CMS (Independent App)
│   │   ├── src/
│   │   │   ├── pages/
│   │   │   │   ├── Inventory/
│   │   │   │   ├── Suppliers/
│   │   │   │   ├── Cashflow/
│   │   │   │   └── Reports/
│   │   │   └── App.tsx
│   │   ├── package.json
│   │   └── vercel.json          # Deploy to cms-umkm.yourdomain.com
│   │
│   ├── cms-farmasi/             # ← Pharmacy CMS (Independent App)
│   │   ├── src/
│   │   │   ├── pages/
│   │   │   │   ├── Prescriptions/
│   │   │   │   ├── Drugs/
│   │   │   │   ├── Patients/
│   │   │   │   └── Compliance/
│   │   │   └── App.tsx
│   │   ├── package.json
│   │   └── vercel.json          # Deploy to cms-farmasi.yourdomain.com
│   │
│   └── cms-manufaktur/          # ← Manufacturing CMS (Independent App)
│       ├── src/
│       │   ├── pages/
│       │   │   ├── Production/
│       │   │   ├── BOM/
│       │   │   ├── QualityControl/
│       │   │   └── Warehouse/
│       │   └── App.tsx
│       ├── package.json
│       └── vercel.json          # Deploy to cms-manufaktur.yourdomain.com
│
├── package.json                 # Root package.json (monorepo)
├── pnpm-workspace.yaml          # Workspace configuration
└── turbo.json                   # Build optimization (optional)
```

### Package Manager Setup (pnpm Workspaces):

```yaml
# pnpm-workspace.yaml
packages:
  - 'packages/*'
```

```json
// package.json (root)
{
  "name": "whatsapp-bot-saas",
  "private": true,
  "workspaces": ["packages/*"],
  "scripts": {
    "dev:core": "pnpm --filter @whatsapp-bot/core dev",
    "dev:saas": "pnpm --filter @whatsapp-bot/cms-saas dev",
    "dev:umkm": "pnpm --filter @whatsapp-bot/cms-umkm dev",
    "dev:farmasi": "pnpm --filter @whatsapp-bot/cms-farmasi dev",
    "build:all": "pnpm -r build",
    "deploy:saas": "pnpm --filter @whatsapp-bot/cms-saas deploy",
    "deploy:umkm": "pnpm --filter @whatsapp-bot/cms-umkm deploy"
  }
}
```

```json
// packages/cms-saas/package.json
{
  "name": "@whatsapp-bot/cms-saas",
  "dependencies": {
    "@whatsapp-bot/core": "workspace:*",
    "react": "^18.0.0",
    "react-router-dom": "^6.0.0"
  }
}
```

### Deployment Strategy:

**Option 1: Separate Subdomains (Recommended)**
```
cms-saas.yourdomain.com       → Vercel Project 1 (cms-saas)
cms-umkm.yourdomain.com       → Vercel Project 2 (cms-umkm)
cms-farmasi.yourdomain.com    → Vercel Project 3 (cms-farmasi)
cms-manufaktur.yourdomain.com → Vercel Project 4 (cms-manufaktur)
```

**Option 2: Path-based Routing**
```
yourdomain.com/cms/saas       → cms-saas
yourdomain.com/cms/umkm       → cms-umkm
yourdomain.com/cms/farmasi    → cms-farmasi
```

### Tenant Access Control:

```typescript
// packages/core/src/auth/Login.tsx
export function Login() {
  const handleLogin = async (phone: string, password: string) => {
    const { user, token } = await authService.login(phone, password);

    // Redirect based on user's subscribed module
    const redirectUrls = {
      saas: 'https://cms-saas.yourdomain.com',
      umkm: 'https://cms-umkm.yourdomain.com',
      farmasi: 'https://cms-farmasi.yourdomain.com',
      manufaktur: 'https://cms-manufaktur.yourdomain.com'
    };

    // Redirect to appropriate CMS
    window.location.href = `${redirectUrls[user.module]}?token=${token}`;
  };
}
```

### Benefits of This Architecture:

| Feature | Monorepo Multi-CMS | Single CMS |
|---------|-------------------|------------|
| **Tenant Isolation** | ✅ Perfect - Separate apps | ❌ Need permission checks |
| **Module-Specific Features** | ✅ Independent codebases | ⚠️ Complex conditionals everywhere |
| **Code Reuse (UI)** | ✅ Via `@whatsapp-bot/core` | ✅ Built-in |
| **Deployment** | ✅ Independent per module | ❌ All or nothing |
| **Scaling** | ✅ Scale per module | ❌ Scale everything |
| **Bundle Size** | ✅ Smaller (module-specific) | ❌ Larger (all modules) |
| **Team Structure** | ✅ Team per module | ❌ Shared team |
| **Feature Development** | ✅ Parallel development | ⚠️ Merge conflicts |

---

## 📦 Frontend Structure

### Shared Core Package (`packages/core/`)

All CMSs import from `@whatsapp-bot/core`:

```typescript
// Example: Using shared components in cms-saas
import { Button, Table, Modal } from '@whatsapp-bot/core/components';
import { DashboardLayout } from '@whatsapp-bot/core/layouts';
import { useAuth, useAPI } from '@whatsapp-bot/core/hooks';

export function OrdersPage() {
  const { user } = useAuth();
  const { data: orders } = useAPI('/saas/orders');

  return (
    <DashboardLayout>
      <h1>Orders Management</h1>
      <Table data={orders} columns={orderColumns} />
    </DashboardLayout>
  );
}
```

**Core Package Structure:**
```
packages/core/src/
├── components/              # Reusable UI components
│   ├── Button/
│   ├── Table/
│   ├── Modal/
│   ├── Form/
│   ├── Chart/
│   └── index.ts
├── layouts/                 # Layout components
│   ├── DashboardLayout.tsx
│   ├── Sidebar.tsx
│   ├── Header.tsx
│   └── index.ts
├── hooks/                   # Custom React hooks
│   ├── useAuth.ts
│   ├── useAPI.ts
│   ├── useToast.ts
│   ├── usePermissions.ts
│   └── index.ts
├── services/                # API services
│   ├── apiClient.ts
│   ├── authService.ts
│   ├── whatsappService.ts
│   └── index.ts
├── utils/                   # Utility functions
│   ├── formatter.ts
│   ├── validation.ts
│   ├── constants.ts
│   └── index.ts
├── types/                   # Shared TypeScript types
│   ├── User.ts
│   ├── Client.ts
│   ├── API.ts
│   └── index.ts
└── index.ts                 # Main export
```

### Module-Specific CMSs

Each CMS is an independent React/Next.js application with module-specific features.

#### CMS SaaS (`packages/cms-saas/`)
```
src/
├── pages/
│   ├── Dashboard/           # Order stats, revenue charts
│   ├── Orders/              # Order list, detail, confirm, cancel
│   ├── Products/            # Product CRUD
│   ├── Customers/           # Customer management
│   ├── Cart/                # Active carts, abandoned carts
│   ├── Analytics/           # Sales analytics
│   └── Marketing/           # Promotions, campaigns
├── components/              # SaaS-specific components
├── hooks/                   # SaaS-specific hooks
└── App.tsx
```

#### CMS UMKM (`packages/cms-umkm/`)
```
src/
├── pages/
│   ├── Dashboard/           # Cashflow summary, profit/loss
│   ├── Inventory/           # Stock management, stock opname
│   ├── Suppliers/           # Supplier directory, purchase orders
│   ├── Cashflow/            # Income/expense tracking
│   ├── Loans/               # Loan management, installment tracking
│   ├── Accounting/          # Basic bookkeeping
│   └── Reports/             # Financial reports
├── components/
├── hooks/
└── App.tsx
```

#### CMS Farmasi (`packages/cms-farmasi/`)
```
src/
├── pages/
│   ├── Dashboard/           # Prescription stats, drug alerts
│   ├── Prescriptions/       # Prescription management
│   ├── Drugs/               # Drug inventory with expiry tracking
│   ├── Patients/            # Patient records
│   ├── Doctors/             # Doctor directory
│   ├── Compliance/          # Regulatory compliance, reporting
│   ├── Alerts/              # Expiry alerts, low stock alerts
│   └── Reports/             # Pharmacy reports
├── components/
├── hooks/
└── App.tsx
```

#### CMS Manufaktur (`packages/cms-manufaktur/`)
```
src/
├── pages/
│   ├── Dashboard/           # Production KPIs
│   ├── Production/          # Production orders, scheduling
│   ├── BOM/                 # Bill of Materials
│   ├── QualityControl/      # QC inspection, defect tracking
│   ├── Warehouse/           # Raw materials inventory
│   ├── Suppliers/           # Supplier management
│   ├── Maintenance/         # Machine maintenance schedules
│   └── Reports/             # Production reports
├── components/
├── hooks/
└── App.tsx
```

---

## 🌐 Backend API Base URL

```
Development: http://localhost:8080
Production: https://your-domain.com
Swagger UI: http://localhost:8080/swagger/
```

---

## 🏗️ Features & Modules

### 1. **Client/Tenant Management**
Multi-tenant system where each client (business) has their own WhatsApp bot.

**Database Table:** `clients`

**Fields:**
- `id` (UUID)
- `whatsapp_number` (Bot's WhatsApp number)
- `business_name`
- `module` (saas, umkm, farmasi, manufacturing)
- `subscription_plan` (free, pro, enterprise)
- `subscription_status` (active, inactive, suspended)
- `created_at`, `updated_at`

**CMS Pages Needed:**
- ✅ Tenant List/Dashboard
- ✅ Tenant Detail/Edit
- ✅ Tenant Creation Form
- ✅ Subscription Management

---

### 2. **User Management (Company Users)**
Admin and staff users for each tenant with role-based access.

**Database Table:** `company_users`

**Fields:**
- `id` (UUID)
- `client_id` (Foreign key to clients)
- `phone_number`
- `name`
- `role` (super_admin, admin_tenant, staff_tenant, customer)

**Roles:**
- `super_admin`: SaaS platform owner (full access)
- `admin_tenant`: Business owner (tenant admin)
- `staff_tenant`: Business staff
- `customer`: End customer (no CMS access)

**CMS Pages Needed:**
- ✅ User List (per tenant)
- ✅ Add/Edit User
- ✅ Role Management
- ✅ User Activity Log

---

### 3. **WhatsApp Bot Management**

**WAHA Integration** - Manage WhatsApp sessions and bot status.

**CMS Pages Needed:**

#### 3.1 Session Management
- **Start Session:** `POST /whatsapp/session/start`
- **Stop Session:** `POST /whatsapp/session/stop`
- **Restart Session:** `POST /whatsapp/session/restart`
- **Get Status:** `GET /whatsapp/session/status`
- **Get QR Code:** `GET /whatsapp/qr` (for initial connection)

**UI Components:**
```
┌─────────────────────────────────────┐
│ WhatsApp Bot Status                 │
├─────────────────────────────────────┤
│ Status: 🟢 Connected                │
│ Number: +62 831-3957-3494           │
│ Session: default                    │
│                                     │
│ [Stop Session] [Restart] [View QR] │
└─────────────────────────────────────┘
```

#### 3.2 Webhook Configuration
- **Configure Webhook:** `POST /whatsapp/webhook/configure`
  ```json
  {
    "webhook_url": "https://your-api.com/webhook"
  }
  ```

---

### 4. **Knowledge Base Management**

Manage FAQ and knowledge items for AI bot responses.

**Database Table:** `knowledge_bases`

**Fields:**
- `id` (UUID)
- `client_id`
- `question`
- `answer`
- `category` (optional)
- `created_at`, `updated_at`

**API Endpoints:**
- `GET /knowledge-base` - List all KB items
- `POST /knowledge-base` - Add new KB item
  ```json
  {
    "client_id": "uuid",
    "question": "Jam operasional?",
    "answer": "Buka Senin-Jumat 08:00-17:00",
    "category": "Info Umum"
  }
  ```

**CMS Pages Needed:**
- ✅ Knowledge Base List (table with search)
- ✅ Add/Edit KB Item Form
- ✅ Bulk Import (CSV)
- ✅ Category Management

**UI Example:**
```
┌────────────────────────────────────────────┐
│ Knowledge Base                 [+ Add New] │
├────────────────────────────────────────────┤
│ Search: [________________] [Filter ▼]      │
├────────────────────────────────────────────┤
│ Question              | Answer      | Edit │
│ Jam operasional?      | Buka Senin…│ ✏️ 🗑️│
│ Cara order?           | Ketik menu…│ ✏️ 🗑️│
└────────────────────────────────────────────┘
```

---

### 5. **Order Management System**

Complete e-commerce order management integrated with WhatsApp.

**Database Table:** `saas_orders`

**Order Model:**
```json
{
  "id": "uuid",
  "client_id": "uuid",
  "order_number": "ORD-20251130-9717",
  "customer_phone": "6287872871856",
  "customer_name": "John Doe",
  "items": [
    {
      "product_id": "PROD001",
      "product_name": "Kopi Arabica",
      "quantity": 2,
      "price": 45000,
      "subtotal": 90000
    }
  ],
  "total_amount": 90000,
  "payment_method": "transfer",
  "payment_status": "pending",
  "payment_gateway": "manual",
  "payment_link": "",
  "payment_reference": "",
  "paid_at": null,
  "fulfillment_status": "pending",
  "created_at": "2025-11-30T10:00:00Z",
  "updated_at": "2025-11-30T10:00:00Z"
}
```

**Payment Status:**
- `pending` - Menunggu pembayaran
- `paid` - Sudah dibayar
- `failed` - Pembayaran gagal
- `cancelled` - Dibatalkan
- `refunded` - Dikembalikan

**Fulfillment Status:**
- `pending` - Belum diproses
- `processing` - Sedang diproses
- `shipped` - Sudah dikirim
- `delivered` - Sudah diterima
- `cancelled` - Dibatalkan

**API Endpoints:**

#### List Orders
```
GET /orders?page=1&limit=10&status=pending&customer_phone=628xxx
```

#### Get Order Detail
```
GET /orders/:id
GET /orders/status/:orderNumber
```

#### Create Order
```
POST /orders
{
  "client_id": "uuid",
  "customer_phone": "628xxx",
  "customer_name": "John Doe",
  "items": [...],
  "total_amount": 90000,
  "payment_method": "transfer"
}
```

#### Update Order
```
PUT /orders/:id
{
  "payment_status": "paid",
  "fulfillment_status": "processing"
}
```

#### Confirm Payment (Manual)
```
POST /orders/:id/confirm-payment
{
  "payment_method": "transfer",
  "reference": "TRF20231130123456",
  "notes": "Transfer dari BCA a/n John Doe"
}
```
**Effect:**
- Updates payment_status to "paid"
- Sends WhatsApp notification to customer
- Updates paid_at timestamp

#### Cancel Order
```
POST /orders/:id/cancel
{
  "reason": "Stok habis"
}
```
**Effect:**
- Updates payment_status to "cancelled"
- Sends friendly cancellation message to customer via WhatsApp
- Example: "😔 Mohon Maaf - Pesanan Anda #ORD-xxx telah dibatalkan. Alasan: Stok habis"

**CMS Pages Needed:**

#### 5.1 Order Dashboard
```
┌─────────────────────────────────────────────────┐
│ Orders Dashboard                                 │
├─────────────────────────────────────────────────┤
│ 📊 Stats                                         │
│ ┌──────┐ ┌──────┐ ┌──────┐ ┌──────┐            │
│ │ 45   │ │ 12   │ │ 8    │ │ 2    │            │
│ │Total │ │Pending│ │Paid │ │Cancel│            │
│ └──────┘ └──────┘ └──────┘ └──────┘            │
└─────────────────────────────────────────────────┘
```

#### 5.2 Order List
```
┌────────────────────────────────────────────────────────┐
│ Orders                          [Export CSV] [Filter ▼]│
├────────────────────────────────────────────────────────┤
│ Search: [__________] Status: [All ▼] Date: [Today ▼]  │
├────────────────────────────────────────────────────────┤
│ Order #       │Customer    │Amount │Status │Actions    │
│ ORD-20251130…│ 6287872... │290K   │Pending│[Detail]   │
│ ORD-20251129…│ 6281234... │150K   │Paid   │[Detail]   │
└────────────────────────────────────────────────────────┘
```

#### 5.3 Order Detail Page
```
┌─────────────────────────────────────────────┐
│ Order #ORD-20251130-9717          [Back]    │
├─────────────────────────────────────────────┤
│ 👤 Customer                                 │
│    Name: John Doe                           │
│    Phone: +62 878-7287-1856                 │
│                                             │
│ 📦 Items                                    │
│    • Kopi Arabica x2 = Rp 90,000           │
│    • Gula Aren x1 = Rp 25,000              │
│                                             │
│ 💰 Total: Rp 115,000                       │
│                                             │
│ 📊 Status                                   │
│    Payment: [Pending ▼]                    │
│    Fulfillment: [Pending ▼]                │
│                                             │
│ 🎬 Actions                                  │
│    [✅ Confirm Payment] [❌ Cancel Order]  │
└─────────────────────────────────────────────┘
```

#### 5.4 Payment Confirmation Modal
```
┌─────────────────────────────────────┐
│ Confirm Payment - ORD-20251130-9717 │
├─────────────────────────────────────┤
│ Payment Method:                     │
│ [transfer     ▼]                    │
│                                     │
│ Reference/Transaction ID:           │
│ [_____________________________]     │
│                                     │
│ Notes:                              │
│ [_____________________________]     │
│ [_____________________________]     │
│                                     │
│        [Cancel] [Confirm Payment]   │
└─────────────────────────────────────┘
```

#### 5.5 Order Cancellation Modal
```
┌─────────────────────────────────────┐
│ Cancel Order - ORD-20251130-9717    │
├─────────────────────────────────────┤
│ Reason for cancellation:            │
│ [_____________________________]     │
│ [_____________________________]     │
│                                     │
│ ⚠️ Customer will receive WhatsApp  │
│    notification about cancellation  │
│                                     │
│        [Back] [Cancel Order]        │
└─────────────────────────────────────┘
```

---

### 6. **Shopping Cart System**

Customers can add items to cart before checkout via WhatsApp.

**Database Table:** `saas_carts`

**Cart Model:**
```json
{
  "id": "uuid",
  "customer_phone": "628xxx",
  "client_id": "uuid",
  "items": [
    {
      "product_id": "PROD001",
      "product_name": "Kopi Arabica",
      "quantity": 2,
      "price": 45000,
      "subtotal": 90000
    }
  ],
  "total_amount": 90000,
  "created_at": "2025-11-30T10:00:00Z",
  "updated_at": "2025-11-30T10:00:00Z"
}
```

**API Endpoints:**

```
POST /cart/add         - Add item to cart
PUT /cart/update       - Update item quantity
DELETE /cart/remove    - Remove item from cart
GET /cart              - View cart (by customer_phone)
DELETE /cart/clear     - Clear all cart items
POST /cart/checkout    - Convert cart to order
```

**CMS Pages Needed:**
- ✅ Active Carts List (see which customers have items in cart)
- ✅ Cart Detail View (see cart contents per customer)
- ✅ Abandoned Cart Report (carts not checked out after X days)

---

### 7. **Product Management** (TO BE BUILT)

**Note:** Product management is NOT yet implemented in backend! This needs to be built.

**Suggested Database Table:** `saas_products`

**Suggested Fields:**
```
id (UUID)
client_id (UUID)
product_id (string, unique per tenant)
name
description
price
stock
category
image_url
is_available (boolean)
created_at
updated_at
```

**CMS Pages Needed:**
- ✅ Product List
- ✅ Add/Edit Product
- ✅ Product Categories
- ✅ Stock Management
- ✅ Price Management
- ✅ Product Images Upload

---

### 8. **Workflow Automation**

Create automated workflows for bot responses.

**Database Table:** `workflows`

**Workflow Model:**
```json
{
  "id": "uuid",
  "client_id": "uuid",
  "name": "Auto Response Jam Operasional",
  "trigger": "keyword_match",
  "trigger_config": {
    "keywords": ["jam", "buka", "tutup"]
  },
  "actions": [
    {
      "type": "send_message",
      "message": "Kami buka Senin-Jumat 08:00-17:00"
    }
  ],
  "is_active": true,
  "created_at": "2025-11-30T10:00:00Z"
}
```

**API Endpoints:**
```
GET /workflows              - List workflows
POST /workflows             - Create workflow
GET /workflows/:id          - Get workflow detail
PUT /workflows/:id          - Update workflow
DELETE /workflows/:id       - Delete workflow
POST /workflows/:id/execute - Execute workflow manually
GET /workflows/:id/executions - Get execution history
```

**CMS Pages Needed:**
- ✅ Workflow List
- ✅ Workflow Builder (drag-and-drop if possible)
- ✅ Workflow Execution Logs

---

### 9. **Analytics & Reports**

**CMS Pages Needed:**

#### 9.1 Overview Dashboard
```
┌─────────────────────────────────────────┐
│ Dashboard - Last 30 Days                │
├─────────────────────────────────────────┤
│ 📈 Total Orders: 156                    │
│ 💰 Revenue: Rp 15,600,000              │
│ 👥 Active Customers: 89                 │
│ 📨 Messages Handled: 1,234             │
│                                         │
│ [Revenue Chart]                         │
│ [Order Status Pie Chart]                │
└─────────────────────────────────────────┘
```

#### 9.2 Customer Report
```
GET /orders/customer?phone=628xxx
```
Show:
- Customer order history
- Total spending
- Last order date
- Favorite products

#### 9.3 Sales Report
- Daily/weekly/monthly sales
- Best selling products
- Revenue trends
- Payment method distribution

---

### 10. **Notification System**

**Admin Notifications:**
System sends WhatsApp notifications to admin when:
- New order received
- Payment confirmed
- Order cancelled

**Configuration:**
```
.env:
ADMIN_PHONE=6281234567890  (Super admin - receives all notifications)
```

**Customer Notifications:**
Sent automatically via WhatsApp:
- Order confirmation
- Payment confirmation
- Order cancellation (with custom reason)
- Payment link (if using payment gateway)

**CMS Pages Needed:**
- ✅ Notification Settings (configure admin phones)
- ✅ Notification Templates (customize message templates)
- ✅ Notification Log (see sent notifications)

---

### 11. **OCR Receipt Processing** (Optional Feature)

Process receipt images via OCR and extract transaction data.

**API Endpoints:**
```
POST /ocr/process-receipt
GET /transactions
```

**CMS Pages Needed:**
- ✅ OCR Transaction List
- ✅ Manual Review/Edit OCR Results

---

## 📊 Database Schema

### Core Tables:

#### `clients` (Tenants)
```sql
id                  UUID PRIMARY KEY
whatsapp_number     TEXT
business_name       TEXT NOT NULL
module              TEXT DEFAULT 'saas'
subscription_plan   TEXT DEFAULT 'free'
subscription_status TEXT DEFAULT 'active'
created_at          TIMESTAMP
updated_at          TIMESTAMP
```

#### `company_users` (Admin/Staff)
```sql
id            UUID PRIMARY KEY
client_id     UUID REFERENCES clients(id)
phone_number  TEXT NOT NULL
name          TEXT
role          TEXT CHECK (role IN ('super_admin', 'admin_tenant', 'staff_tenant', 'customer'))
created_at    TIMESTAMP
UNIQUE(client_id, phone_number)
```

#### `saas_orders`
```sql
id                   UUID PRIMARY KEY
client_id            UUID NOT NULL
order_number         TEXT UNIQUE NOT NULL
customer_phone       TEXT NOT NULL
customer_name        TEXT
items                JSONB NOT NULL
total_amount         DECIMAL(12,2) NOT NULL
payment_method       TEXT
payment_status       TEXT DEFAULT 'pending'
payment_gateway      TEXT
payment_link         TEXT
payment_reference    TEXT
paid_at              TIMESTAMP
fulfillment_status   TEXT DEFAULT 'pending'
created_at           TIMESTAMP
updated_at           TIMESTAMP
```

#### `saas_carts`
```sql
id             UUID PRIMARY KEY
customer_phone TEXT NOT NULL
client_id      UUID NOT NULL
items          JSONB NOT NULL
total_amount   DECIMAL(12,2) DEFAULT 0
created_at     TIMESTAMP
updated_at     TIMESTAMP
deleted_at     TIMESTAMP
UNIQUE(customer_phone, client_id)
```

#### `knowledge_bases`
```sql
id         UUID PRIMARY KEY
client_id  UUID REFERENCES clients(id)
question   TEXT NOT NULL
answer     TEXT NOT NULL
category   TEXT
created_at TIMESTAMP
updated_at TIMESTAMP
```

#### `workflows`
```sql
id              UUID PRIMARY KEY
client_id       UUID REFERENCES clients(id)
name            TEXT NOT NULL
trigger         TEXT NOT NULL
trigger_config  JSONB
actions         JSONB NOT NULL
is_active       BOOLEAN DEFAULT true
created_at      TIMESTAMP
updated_at      TIMESTAMP
```

---

## 👥 User Roles & Permissions

### `super_admin` (SaaS Owner)
**Access:**
- ✅ All tenants/clients
- ✅ Create/edit/delete tenants
- ✅ View all orders across tenants
- ✅ Manage super admin users
- ✅ System configuration
- ✅ Analytics across all tenants

### `admin_tenant` (Business Owner)
**Access:**
- ✅ Own tenant only
- ✅ View/manage orders
- ✅ Confirm/cancel orders
- ✅ Manage knowledge base
- ✅ Manage workflows
- ✅ Manage staff users
- ✅ WhatsApp bot settings
- ✅ Analytics for own tenant
- ❌ Cannot access other tenants

### `staff_tenant` (Business Staff)
**Access:**
- ✅ View orders
- ✅ Update order status
- ✅ View knowledge base
- ❌ Cannot edit settings
- ❌ Cannot manage users
- ❌ Cannot delete data

### `customer` (End Customer)
**Access:**
- ❌ No CMS access
- ✅ WhatsApp bot only

---

## 🚀 Implementation Priority

### Phase 1: Core CMS (HIGH PRIORITY)
1. ✅ **Authentication & Authorization**
   - Login page
   - Role-based access control
   - Session management

2. ✅ **Dashboard**
   - Order statistics
   - Revenue overview
   - Recent activities

3. ✅ **Order Management**
   - Order list
   - Order detail
   - Confirm payment
   - Cancel order
   - Status updates

4. ✅ **Tenant Management** (for super_admin)
   - Tenant list
   - Add/edit tenant
   - Subscription management

5. ✅ **User Management**
   - User list
   - Add/edit users
   - Role assignment

### Phase 2: Bot Management (MEDIUM PRIORITY)
6. ✅ **WhatsApp Bot Control**
   - Session status
   - Start/stop session
   - QR code display
   - Webhook configuration

7. ✅ **Knowledge Base**
   - KB item list
   - Add/edit KB items
   - Bulk import

### Phase 3: Advanced Features (LOW PRIORITY)
8. ✅ **Product Management** (needs backend implementation first)
9. ✅ **Workflow Builder**
10. ✅ **Analytics & Reports**
11. ✅ **Notification Management**
12. ✅ **OCR Transaction Review**

---

## 🔗 API Integration Examples

### Authentication
```javascript
// Note: Backend doesn't have auth yet - implement JWT or session-based auth

// Suggested implementation:
POST /auth/login
{
  "phone_number": "6285224111826",
  "password": "xxx"  // Or use OTP
}

Response:
{
  "token": "jwt_token",
  "user": {
    "id": "uuid",
    "name": "Admin",
    "role": "admin_tenant",
    "client_id": "uuid"
  }
}
```

### Fetch Orders
```javascript
const fetchOrders = async (status = '', page = 1) => {
  const response = await fetch(
    `http://localhost:8080/orders?status=${status}&page=${page}&limit=20`,
    {
      headers: {
        'Authorization': `Bearer ${token}`  // If auth implemented
      }
    }
  );
  const data = await response.json();
  return data;
};
```

### Confirm Payment
```javascript
const confirmPayment = async (orderId, paymentData) => {
  const response = await fetch(
    `http://localhost:8080/orders/${orderId}/confirm-payment`,
    {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json'
      },
      body: JSON.stringify({
        payment_method: paymentData.method,
        reference: paymentData.reference,
        notes: paymentData.notes
      })
    }
  );
  return await response.json();
};
```

### Cancel Order
```javascript
const cancelOrder = async (orderId, reason) => {
  const response = await fetch(
    `http://localhost:8080/orders/${orderId}/cancel`,
    {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json'
      },
      body: JSON.stringify({ reason })
    }
  );
  return await response.json();
};
```

---

## 📝 Notes & Recommendations

### Backend Gaps (Need to be built):
1. ❌ **Authentication System** - No JWT/session auth yet
2. ❌ **Product Management** - No product CRUD APIs
3. ❌ **Staff permission middleware** - No endpoint-level authorization
4. ❌ **File upload** - No image upload for products
5. ❌ **Pagination** - Some endpoints don't have pagination

### Frontend Best Practices:
1. ✅ Use React Query or SWR for data fetching
2. ✅ Implement optimistic updates for better UX
3. ✅ Add loading states and error handling
4. ✅ Use toast notifications for success/error feedback
5. ✅ Implement proper form validation
6. ✅ Add confirmation dialogs for destructive actions
7. ✅ Make it mobile-responsive (admin might use on phone)

### Security Considerations:
1. ⚠️ Add CSRF protection
2. ⚠️ Implement rate limiting
3. ⚠️ Add input validation on frontend
4. ⚠️ Sanitize user inputs
5. ⚠️ Use HTTPS in production
6. ⚠️ Implement proper error messages (don't leak system info)

---

## 🎨 UI/UX Suggestions

### Design System:
- Use a component library: Material-UI, Ant Design, or Chakra UI
- Consistent color scheme (primary, secondary, success, error)
- Clear typography hierarchy
- Accessible (WCAG 2.1 compliant)

### Key UI Components:
- Data tables with sorting, filtering, pagination
- Modal dialogs for forms
- Toast notifications
- Loading spinners
- Empty states
- Error states
- Confirmation dialogs
- Breadcrumbs for navigation
- Sidebar navigation

---

## 📞 Support & Questions

For backend API questions, check:
- Swagger UI: `http://localhost:8080/swagger/`
- Source code: `/internal/modules/saas/handlers/`

**Key Backend Files:**
- Order logic: `internal/modules/saas/services/order_service.go`
- Payment handling: `internal/modules/saas/handlers/payment_handler.go`
- Admin commands: `internal/modules/saas/services/webhook_service_admin.go`
- Models: `internal/modules/saas/models/`

---

## 🚦 Quick Start Checklist for Frontend Dev

- [ ] Setup React/Next.js project
- [ ] Configure API base URL
- [ ] Implement authentication (coordinate with backend for JWT implementation)
- [ ] Create layout with sidebar navigation
- [ ] Build Order List page
- [ ] Build Order Detail page
- [ ] Implement Confirm Payment modal
- [ ] Implement Cancel Order modal
- [ ] Test order management flow end-to-end
- [ ] Add tenant switching (for super_admin)
- [ ] Implement dashboard with statistics
- [ ] Build Knowledge Base management
- [ ] Add WhatsApp bot controls

---

**Last Updated:** 2025-12-01
**Backend Version:** v1.0
**Contact:** Backend team for API changes and new endpoints
