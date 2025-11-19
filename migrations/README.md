# Database Migrations

This directory contains database migrations for the micro-system-ai-agent backend.

## Structure

```
migrations/
├── saas/           # SAAS module migrations
│   ├── 000001_create_saas_clients.up.sql
│   ├── 000001_create_saas_clients.down.sql
│   ├── 000002_create_saas_knowledge_base.up.sql
│   └── ...
├── umkm/           # UMKM module migrations (future)
└── farmasi/        # Farmasi module migrations (future)
```

## Running Migrations

### Using Makefile (Recommended)

```bash
# Run UP migrations for SAAS module
make migrate-up MODULE=saas

# Run DOWN migrations for SAAS module
make migrate-down MODULE=saas

# Check current migration version
make migrate-version MODULE=saas

# Force migration to specific version (emergency only)
make migrate-force MODULE=saas VERSION=1
```

### Using Go Command Directly

```bash
# UP migrations
go run cmd/migrate/main.go -module=saas -cmd=up

# DOWN migrations
go run cmd/migrate/main.go -module=saas -cmd=down

# Check version
go run cmd/migrate/main.go -module=saas -cmd=version
```

### Using migrate CLI directly

```bash
# Install migrate CLI
go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest

# Run migrations
migrate -path migrations/saas -database "postgresql://user:pass@host:port/db" up
```

## Creating New Migrations

```bash
# Create a new migration file
migrate create -ext sql -dir migrations/saas -seq add_new_field

# This will create:
# - migrations/saas/000005_add_new_field.up.sql
# - migrations/saas/000005_add_new_field.down.sql
```

## Migration Guidelines

1. **Always create both UP and DOWN migrations**
2. **Test migrations locally before deploying**
3. **Never edit existing migration files** - create a new one instead
4. **Use transactions when possible** - PostgreSQL supports DDL in transactions
5. **Add indexes for foreign keys and frequently queried columns**
6. **Use meaningful migration names** - describe what it does

## Schema Overview

### saas_clients
- Core client information
- Subscription management
- WhatsApp integration

### saas_knowledge_base
- **JSONB-based flexible content** - supports FAQ, products, services, policies
- Rich text editor content auto-converted to JSON
- Full-text search capable
- Tag-based organization

### saas_conversations
- Customer interaction history
- Message tracking
- AI response logging

### saas_credits
- Usage tracking
- Credit management per client
- Billing periods

## Troubleshooting

### Migration stuck in dirty state

```bash
# Check version
make migrate-version MODULE=saas

# Force to specific version
make migrate-force MODULE=saas VERSION=3
```

### Migration fails

1. Check database connection in `.env`
2. Verify SQL syntax in migration file
3. Check logs for specific error
4. Rollback: `make migrate-down MODULE=saas`

## Best Practices

- **Development**: Run migrations locally first
- **Staging**: Test migrations before production
- **Production**: Always backup database before migrating
- **Rollback Plan**: Always have a DOWN migration ready
