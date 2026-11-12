package knowledgebases

import (
	"database/sql"
	"fmt"
	"hash/fnv"
	"math"
	"sort"
	"strings"
)

// EmbeddingDims is this package's own disclosed default for the
// quantized embedding vector's fixed dimensionality - the
// specification names no concrete size for "quantized embeddings,"
// so this is this project's own chosen value (the same "disclosed
// default, not a value the specification gives" posture every other
// fixed constant in this codebase already takes).
const EmbeddingDims = 256

// EnsureEmbeddingSchema creates the knowledge_base_embedding table if
// it does not already exist - "Local indexing for retrieval"'s
// quantized-embeddings half ("The system SHALL index attached bases
// locally using both textual search and quantized embeddings,
// without sending base content to a remote service"), alongside
// [EnsureSearchSchema]'s own textual-search half.
//
// Honesty note: [Embed] computes a classical, hashed bag-of-words
// vector (the "hashing trick" - Weinberger et al., feature hashing),
// not a learned neural embedding. A real semantic-embedding model
// needs either a sizable bundled/downloaded model or CGO bindings to
// run one locally (ONNX Runtime, llama.cpp) - neither fits this
// codebase's own established no-CGO, no-bundled-model-weights posture
// (modernc.org/sqlite is pure Go for the same reason everywhere else
// in this codebase reads an OS or file resource). This is still a
// genuine, local, content-derived vector representation, and it is
// genuinely quantized (int8) - what the specification's own two
// concrete words actually require - not a claim of semantic
// understanding this package cannot deliver.
func EnsureEmbeddingSchema(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS knowledge_base_embedding (
			base_id   INTEGER NOT NULL,
			file_path TEXT NOT NULL,
			vector    BLOB NOT NULL,
			PRIMARY KEY (base_id, file_path)
		);
	`)
	if err != nil {
		return fmt.Errorf("knowledgebases: creating embedding schema: %w", err)
	}
	return nil
}

// Embed computes text's own quantized hashed-bag-of-words vector,
// [EmbeddingDims] int8 components long - see [EnsureEmbeddingSchema]'s
// own doc comment for what this is and is not. Each lowercased word
// token hashes (FNV-1a) to one of EmbeddingDims buckets, signed by one
// more hash bit (the standard feature-hashing trick: signed hashing
// cancels out collision bias better than always adding), accumulated
// as float64, L2-normalized so document length does not dominate
// magnitude, then quantized to int8 by scaling into [-127, 127] (an
// L2-normalized vector's own components are each already within
// [-1, 1], so this scaling never overflows int8's range).
func Embed(text string) []int8 {
	buckets := make([]float64, EmbeddingDims)
	for _, token := range strings.Fields(strings.ToLower(text)) {
		h := fnv.New64a()
		_, _ = h.Write([]byte(token))
		sum := h.Sum64()
		bucket := int(sum % uint64(EmbeddingDims))
		sign := 1.0
		if (sum>>1)&1 == 1 {
			sign = -1.0
		}
		buckets[bucket] += sign
	}

	var sumSquares float64
	for _, v := range buckets {
		sumSquares += v * v
	}
	norm := math.Sqrt(sumSquares)

	vector := make([]int8, EmbeddingDims)
	if norm == 0 {
		return vector // an empty, or sign-canceling, text embeds to the zero vector
	}
	for i, v := range buckets {
		vector[i] = int8(math.Round((v / norm) * 127))
	}
	return vector
}

// storeEmbedding persists baseID's own quantized embedding for the
// file at path, computed from content - called from [IndexBase]'s
// own per-file walk, reusing the same content already read for the
// textual index rather than reading the file a second time.
func storeEmbedding(db *sql.DB, baseID int64, path string, content []byte) error {
	vector := Embed(string(content))
	raw := make([]byte, len(vector))
	for i, v := range vector {
		raw[i] = byte(v)
	}
	_, err := db.Exec(
		`INSERT INTO knowledge_base_embedding (base_id, file_path, vector) VALUES (?, ?, ?)
		 ON CONFLICT(base_id, file_path) DO UPDATE SET vector = excluded.vector`,
		baseID, path, raw,
	)
	if err != nil {
		return fmt.Errorf("knowledgebases: storing embedding for %q: %w", path, err)
	}
	return nil
}

// EmbeddingSearch returns every indexed file's own [SearchResult],
// most similar to query first by cosine similarity between query's
// own [Embed]ded vector and each file's stored one - the quantized-
// embeddings counterpart to [Search]'s own textual full-text search.
// Snippet is always empty: a cosine-similarity ranking has no matched
// substring to excerpt, unlike FTS5's own snippet() function.
//
// Unlike [Search] (SQLite's own FTS5 MATCH, which only returns rows
// that actually match), this ranks every indexed file, including ones
// with zero similarity to query - a caller wanting only the closest
// few should slice the result itself (see [EmbeddingSearchInScope]'s
// own doc comment on why this does not cap the count before scope
// filtering).
func EmbeddingSearch(db *sql.DB, query string) ([]SearchResult, error) {
	queryVector := Embed(query)

	rows, err := db.Query(`SELECT base_id, file_path, vector FROM knowledge_base_embedding`)
	if err != nil {
		return nil, fmt.Errorf("knowledgebases: searching embeddings: %w", err)
	}
	defer func() { _ = rows.Close() }()

	type scored struct {
		result     SearchResult
		similarity float64
	}
	var all []scored
	for rows.Next() {
		var baseID int64
		var filePath string
		var raw []byte
		if err := rows.Scan(&baseID, &filePath, &raw); err != nil {
			return nil, fmt.Errorf("knowledgebases: searching embeddings: %w", err)
		}
		vector := make([]int8, len(raw))
		for i, b := range raw {
			vector[i] = int8(b)
		}
		all = append(all, scored{
			result:     SearchResult{BaseID: baseID, FilePath: filePath},
			similarity: cosineSimilarity(queryVector, vector),
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("knowledgebases: searching embeddings: %w", err)
	}

	sort.SliceStable(all, func(i, j int) bool { return all[i].similarity > all[j].similarity })
	results := make([]SearchResult, len(all))
	for i, s := range all {
		results[i] = s.result
	}
	return results, nil
}

// EmbeddingSearchInScope is [EmbeddingSearch], filtered to only bases
// [InScope] for area - the embedding counterpart to [SearchInScope].
// It ranks every indexed file first and filters by scope after, the
// same order [SearchInScope] itself already uses, so a base outside
// area never crowds out an in-scope one further down the full
// similarity ranking.
func EmbeddingSearchInScope(db *sql.DB, query, area string) ([]SearchResult, error) {
	results, err := EmbeddingSearch(db, query)
	if err != nil {
		return nil, err
	}

	bases, err := List(db)
	if err != nil {
		return nil, err
	}
	inScope := make(map[int64]bool, len(bases))
	for _, b := range bases {
		inScope[b.ID] = InScope(b, area)
	}

	var filtered []SearchResult
	for _, r := range results {
		if inScope[r.BaseID] {
			filtered = append(filtered, r)
		}
	}
	return filtered, nil
}

// cosineSimilarity reports the cosine similarity between a and b,
// two quantized embedding vectors of possibly different lengths
// (compared only up to the shorter one) - 1.0 for identical
// direction, 0.0 for orthogonal or either vector being all-zero (a
// zero vector has no direction to compare).
func cosineSimilarity(a, b []int8) float64 {
	n := len(a)
	if len(b) < n {
		n = len(b)
	}

	var dot, normA, normB float64
	for i := 0; i < n; i++ {
		fa, fb := float64(a[i]), float64(b[i])
		dot += fa * fb
		normA += fa * fa
		normB += fb * fb
	}
	if normA == 0 || normB == 0 {
		return 0
	}
	return dot / (math.Sqrt(normA) * math.Sqrt(normB))
}
