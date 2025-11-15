package llm

import (
	"fmt"
	"strings"
)

type KnowledgeBase struct {
	BusinessName string
	Tone         string
	FAQs         []FAQ
	Products     []Product
}

type FAQ struct {
	Question string
	Answer   string
}

type Product struct {
	Name  string
	Price float64
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

	sb.WriteString("Instruksi:\n")
	sb.WriteString("- Jawab dengan ramah dan profesional\n")
	sb.WriteString("- Gunakan informasi di atas untuk menjawab pertanyaan\n")
	sb.WriteString("- Jika tidak tahu, katakan dengan jujur\n")
	sb.WriteString("- Jangan membuat informasi yang tidak ada\n")

	return sb.String()
}
