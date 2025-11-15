package tenant

import (
	"database/sql"
	"fmt"
)

type TenantContext struct {
	CompanyID string
	Module    string // "saas", "farmasi", "umkm"
	Role      string // "customer", "admin", "staff"
	ClientID  string // ID dari table clients (untuk backward compatibility)
}

type Resolver struct {
	db *sql.DB
}

func NewResolver(db *sql.DB) *Resolver {
	return &Resolver{db: db}
}

// ResolveFromPhone menentukan company_id, module, dan role dari nomor WA
func (r *Resolver) ResolveFromPhone(phoneNumber string) (*TenantContext, error) {
	// Format: hapus prefix +, ambil nomor saja
	cleanPhone := phoneNumber
	if len(cleanPhone) > 0 && cleanPhone[0] == '+' {
		cleanPhone = cleanPhone[1:]
	}

	ctx := &TenantContext{}

	// Query 1: Cek apakah ini admin/staff dari company
	query := `
		SELECT cu.company_id, c.module, cu.role, cu.client_id
		FROM company_users cu
		JOIN clients c ON c.id = cu.client_id
		WHERE cu.phone_number = $1 AND c.subscription_status = 'active'
		LIMIT 1
	`
	err := r.db.QueryRow(query, cleanPhone).Scan(&ctx.CompanyID, &ctx.Module, &ctx.Role, &ctx.ClientID)
	if err == nil {
		return ctx, nil
	}

	// Query 2: Jika tidak ketemu, cek di table clients (backward compatibility)
	// Ini untuk skenario lama dimana whatsapp_number ada di table clients
	queryLegacy := `
		SELECT id, 'saas' as module, 'admin' as role, id as client_id
		FROM clients
		WHERE whatsapp_number = $1 AND subscription_status = 'active'
		LIMIT 1
	`
	err = r.db.QueryRow(queryLegacy, cleanPhone).Scan(&ctx.CompanyID, &ctx.Module, &ctx.Role, &ctx.ClientID)
	if err == nil {
		return ctx, nil
	}

	// Query 3: Jika masih tidak ketemu, treat as customer
	// Ambil client pertama yang aktif (untuk demo/testing)
	// CATATAN: Di production, ini harus lebih smart (misal dari context routing)
	queryDefault := `
		SELECT id, 'saas' as module, id as client_id
		FROM clients
		WHERE subscription_status = 'active'
		LIMIT 1
	`
	err = r.db.QueryRow(queryDefault).Scan(&ctx.CompanyID, &ctx.Module, &ctx.ClientID)
	if err != nil {
		return nil, fmt.Errorf("no active client found")
	}

	ctx.Role = "customer"
	return ctx, nil
}

// ResolveFromClientID menentukan context dari client_id (untuk API calls)
func (r *Resolver) ResolveFromClientID(clientID string) (*TenantContext, error) {
	ctx := &TenantContext{
		ClientID: clientID,
	}

	query := `
		SELECT id, COALESCE(module, 'saas') as module
		FROM clients
		WHERE id = $1
	`
	err := r.db.QueryRow(query, clientID).Scan(&ctx.CompanyID, &ctx.Module)
	if err != nil {
		return nil, fmt.Errorf("client not found")
	}

	ctx.Role = "admin" // Default untuk API calls
	return ctx, nil
}
