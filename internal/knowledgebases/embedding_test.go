package knowledgebases

import (
	"os"
	"path/filepath"
	"testing"
)

func TestEmbed_Deterministic(t *testing.T) {
	a := Embed("o titular dos dados pode solicitar a exclusao")
	b := Embed("o titular dos dados pode solicitar a exclusao")
	if len(a) != EmbeddingDims || len(b) != EmbeddingDims {
		t.Fatalf("len(a) = %d, len(b) = %d, want %d", len(a), len(b), EmbeddingDims)
	}
	for i := range a {
		if a[i] != b[i] {
			t.Fatalf("Embed is not deterministic: a[%d] = %d, b[%d] = %d", i, a[i], i, b[i])
		}
	}
}

func TestEmbed_EmptyTextIsZeroVector(t *testing.T) {
	v := Embed("")
	for i, c := range v {
		if c != 0 {
			t.Errorf("v[%d] = %d, want 0 (empty text has no tokens)", i, c)
		}
	}
}

func TestEmbed_IdenticalTextIsMoreSimilarThanUnrelatedText(t *testing.T) {
	ref := Embed("o titular dos dados pode solicitar a exclusao")
	same := Embed("o titular dos dados pode solicitar a exclusao")
	unrelated := Embed("receita de bolo de chocolate com cobertura")

	simSame := cosineSimilarity(ref, same)
	simUnrelated := cosineSimilarity(ref, unrelated)
	if simSame <= simUnrelated {
		t.Errorf("cosineSimilarity(identical) = %v, cosineSimilarity(unrelated) = %v, want identical strictly higher", simSame, simUnrelated)
	}
	if simSame < 0.99 {
		t.Errorf("cosineSimilarity(identical text) = %v, want ~1.0", simSame)
	}
}

func TestCosineSimilarity_ZeroVectorIsZero(t *testing.T) {
	zero := make([]int8, EmbeddingDims)
	other := Embed("qualquer coisa")
	if got := cosineSimilarity(zero, other); got != 0 {
		t.Errorf("cosineSimilarity(zero vector, other) = %v, want 0", got)
	}
}

func TestEnsureEmbeddingSchema_IsIdempotent(t *testing.T) {
	db := openSearchTestStore(t) // already calls EnsureEmbeddingSchema once
	if err := EnsureEmbeddingSchema(db); err != nil {
		t.Fatalf("EnsureEmbeddingSchema (second call) returned unexpected error: %v", err)
	}
}

func TestEmbeddingSearch_RanksMostSimilarFirst(t *testing.T) {
	db := openSearchTestStore(t)

	lgpdPath := filepath.Join(t.TempDir(), "lgpd.txt")
	if err := os.WriteFile(lgpdPath, []byte("o titular dos dados pode solicitar a exclusao dos dados"), 0o644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}
	lgpdBase, err := Add(db, Base{Path: lgpdPath, Authority: Reference, Scope: "project"})
	if err != nil {
		t.Fatalf("Add returned unexpected error: %v", err)
	}
	if err := IndexBase(db, lgpdBase); err != nil {
		t.Fatalf("IndexBase returned unexpected error: %v", err)
	}

	bakingPath := filepath.Join(t.TempDir(), "baking.txt")
	if err := os.WriteFile(bakingPath, []byte("receita de bolo de chocolate com cobertura"), 0o644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}
	bakingBase, err := Add(db, Base{Path: bakingPath, Authority: Reference, Scope: "project"})
	if err != nil {
		t.Fatalf("Add returned unexpected error: %v", err)
	}
	if err := IndexBase(db, bakingBase); err != nil {
		t.Fatalf("IndexBase returned unexpected error: %v", err)
	}

	results, err := EmbeddingSearch(db, "o titular dos dados pode solicitar a exclusao")
	if err != nil {
		t.Fatalf("EmbeddingSearch returned unexpected error: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("len(results) = %d, want 2 (every indexed file is ranked)", len(results))
	}
	if results[0].BaseID != lgpdBase.ID {
		t.Errorf("results[0] = %+v, want the lgpd base ranked first (closest match)", results[0])
	}
}

func TestEmbeddingSearchInScope_ExcludesBaseFromADifferentArea(t *testing.T) {
	db := openSearchTestStore(t)

	shippingPath := filepath.Join(t.TempDir(), "shipping.txt")
	if err := os.WriteFile(shippingPath, []byte("a norma de frete fala sobre o limite de peso"), 0o644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}
	shippingBase, err := Add(db, Base{Path: shippingPath, Authority: Reference, Scope: "area:shipping"})
	if err != nil {
		t.Fatalf("Add returned unexpected error: %v", err)
	}
	if err := IndexBase(db, shippingBase); err != nil {
		t.Fatalf("IndexBase returned unexpected error: %v", err)
	}

	billingPath := filepath.Join(t.TempDir(), "billing.txt")
	if err := os.WriteFile(billingPath, []byte("a norma de cobranca tambem fala sobre o limite de peso"), 0o644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}
	billingBase, err := Add(db, Base{Path: billingPath, Authority: Reference, Scope: "area:billing"})
	if err != nil {
		t.Fatalf("Add returned unexpected error: %v", err)
	}
	if err := IndexBase(db, billingBase); err != nil {
		t.Fatalf("IndexBase returned unexpected error: %v", err)
	}

	results, err := EmbeddingSearchInScope(db, "limite de peso", "shipping")
	if err != nil {
		t.Fatalf("EmbeddingSearchInScope returned unexpected error: %v", err)
	}
	for _, r := range results {
		if r.BaseID == billingBase.ID {
			t.Errorf("EmbeddingSearchInScope(..., \"shipping\") included the billing-scoped base: %+v", results)
		}
	}
	found := false
	for _, r := range results {
		if r.BaseID == shippingBase.ID {
			found = true
		}
	}
	if !found {
		t.Errorf("EmbeddingSearchInScope(..., \"shipping\") = %+v, want the shipping-scoped base included", results)
	}
}

func TestIndexBase_ReindexingClearsStaleEmbeddings(t *testing.T) {
	db := openSearchTestStore(t)
	path := filepath.Join(t.TempDir(), "doc.txt")
	if err := os.WriteFile(path, []byte("conteudo original"), 0o644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}
	b, err := Add(db, Base{Path: path, Authority: Reference, Scope: "project"})
	if err != nil {
		t.Fatalf("Add returned unexpected error: %v", err)
	}
	if err := IndexBase(db, b); err != nil {
		t.Fatalf("IndexBase returned unexpected error: %v", err)
	}

	var countBefore int
	if err := db.QueryRow(`SELECT COUNT(*) FROM knowledge_base_embedding WHERE base_id = ?`, b.ID).Scan(&countBefore); err != nil {
		t.Fatalf("counting embeddings failed: %v", err)
	}
	if countBefore != 1 {
		t.Fatalf("countBefore = %d, want 1", countBefore)
	}

	if err := IndexBase(db, b); err != nil {
		t.Fatalf("second IndexBase returned unexpected error: %v", err)
	}
	var countAfter int
	if err := db.QueryRow(`SELECT COUNT(*) FROM knowledge_base_embedding WHERE base_id = ?`, b.ID).Scan(&countAfter); err != nil {
		t.Fatalf("counting embeddings failed: %v", err)
	}
	if countAfter != 1 {
		t.Errorf("countAfter = %d, want 1 (re-indexing should not duplicate embeddings)", countAfter)
	}
}
