package llm

import (
	"encoding/json"
	"fmt"
	"strings"
)

type KnowledgeBase struct {
	BusinessName string
	Tone         string
	FAQs         []FAQ
	Products     []Product
	RawEntries   []RawKBEntry // New: for all other types
}

type FAQ struct {
	Question string
	Answer   string
}

type Product struct {
	Name  string
	Price float64
}

// RawKBEntry represents any KB entry with flexible content
type RawKBEntry struct {
	Type    string                 `json:"type"`
	Title   string                 `json:"title"`
	Content map[string]interface{} `json:"content"`
}

// BuildSystemPrompt membuat system prompt dari knowledge base
func BuildSystemPrompt(kb *KnowledgeBase) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("Anda adalah asisten virtual untuk %s.\n", kb.BusinessName))
	sb.WriteString(fmt.Sprintf("Tone komunikasi: %s.\n\n", kb.Tone))

	// FAQ Section
	if len(kb.FAQs) > 0 {
		sb.WriteString("=== PERTANYAAN UMUM ===\n")
		for _, faq := range kb.FAQs {
			sb.WriteString(fmt.Sprintf("Q: %s\nA: %s\n\n", faq.Question, faq.Answer))
		}
	}

	// Products Section
	if len(kb.Products) > 0 {
		sb.WriteString("=== DAFTAR PRODUK ===\n")
		for _, prod := range kb.Products {
			sb.WriteString(fmt.Sprintf("- %s: Rp %.0f\n", prod.Name, prod.Price))
		}
		sb.WriteString("\n")
	}

	// Raw Entries Section (Services, Policies, Promos, Info, Contact, etc.)
	if len(kb.RawEntries) > 0 {
		sb.WriteString("=== INFORMASI TAMBAHAN ===\n")
		for _, entry := range kb.RawEntries {
			sb.WriteString(fmt.Sprintf("\n**%s** (%s):\n", entry.Title, entry.Type))

			// Convert content to pretty JSON string
			contentJSON, err := json.MarshalIndent(entry.Content, "", "  ")
			if err == nil {
				sb.WriteString(string(contentJSON))
				sb.WriteString("\n")
			}
		}
		sb.WriteString("\n")
	}

	sb.WriteString("Instruksi:\n")
	sb.WriteString("- Kamu adalah asisten yang ramah, helpful, dan NATURAL seperti admin toko\n")
	sb.WriteString("- BOLEH jawab pertanyaan umum/casual (cuaca, tanggal, tips, motivasi, dll) dengan santai dan natural\n")
	sb.WriteString("- Untuk pertanyaan umum: jawab dulu dengan natural, lalu SOFT REDIRECT ke produk/layanan toko\n")
	sb.WriteString("- Untuk pertanyaan produk/layanan: gunakan info dari knowledge base di atas\n")
	sb.WriteString("- Jika ada pertanyaan spesifik yang tidak ada di knowledge base, sarankan kontak langsung\n")
	sb.WriteString("- Maksimal 2-3 kalimat per response, jangan bertele-tele\n")
	sb.WriteString("- Jangan gunakan markdown formatting yang berlebihan\n")
	sb.WriteString("- Berikan improvisasi dan kreativitas dalam jawaban, jangan kaku!\n\n")
	sb.WriteString("Contoh Response yang Baik:\n\n")
	sb.WriteString("User: \"Gimana caranya jadi kaya?\"\n")
	sb.WriteString("Bot: \"Wah pertanyaan bagus! Salah satu caranya ya dengan berbisnis dan jual produk berkualitas. Ngomong-ngomong, mau coba produk kita? Recommended banget lho!\"\n\n")
	sb.WriteString("User: \"Cuaca panas banget hari ini\"\n")
	sb.WriteString("Bot: \"Iya bener nih panas banget ya! Enak tuh kalau sambil nyeruput minuman dingin. Mau coba produk kita? Pas banget buat cuaca gini!\"\n\n")
	sb.WriteString("User: \"Lagi bad mood nih\"\n")
	sb.WriteString("Bot: \"Waduh, semangat ya! Biasanya kalau lagi bad mood enaknya treat yourself dengan sesuatu yang enak. Mau coba produk kita? Bisa jadi mood booster!\"\n\n")
	sb.WriteString("User: \"Hari ini tanggal berapa?\"\n")
	sb.WriteString("Bot: \"Waduh maaf aku ga punya kalender nih hehe. Coba cek di HP kamu aja ya. Btw, ada yang bisa aku bantu terkait produk atau layanan kita?\"\n")

	return sb.String()
}
