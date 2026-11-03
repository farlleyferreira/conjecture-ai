package cjtr

import "fmt"

// VocabularyCategory groups vocabulary entries by what kind of statement
// they appear in.
type VocabularyCategory string

const (
	ComparatorCategory VocabularyCategory = "comparator"
	QuantifierCategory VocabularyCategory = "quantifier"
	DateCategory       VocabularyCategory = "date"
	ArithmeticCategory VocabularyCategory = "arithmetic"
	LogicCategory      VocabularyCategory = "logic"
	CatalogCategory    VocabularyCategory = "catalog_transformation"
	DurationCategory   VocabularyCategory = "duration"
	FrequencyCategory  VocabularyCategory = "frequency"
)

// VocabularyEntry is one fixed phrase (or group of synonymous phrases) of
// the .cjtr controlled vocabulary - the only words a .cjtr statement may
// use besides its own concept names (cjtr-language spec, "Closed
// vocabulary and grammar").
type VocabularyEntry struct {
	// Phrases are the exact English words/phrases carrying this meaning.
	// Some entries have more than one, e.g. "is" and "equals".
	Phrases []string
	// Meaning is the phrase's exact meaning, in English.
	Meaning string
	// PortugueseReading is the phrase's Portuguese reading, as declared in
	// the specification (not a general-purpose translation: it exists so
	// a human reviewer can read a statement in Portuguese without the
	// engine ever compiling that text).
	PortugueseReading string
	Category          VocabularyCategory
}

// Vocabulary is the closed, single source of truth for every phrase a
// .cjtr statement may use (besides a file's own concept names). Anything
// else - a parser, a JSON Schema, autocomplete data, the Portuguese
// translation, or tests - must be generated from this table, never
// maintained as an independent copy.
var Vocabulary = []VocabularyEntry{
	{[]string{"is", "equals"}, "Equal", "é, é igual a", ComparatorCategory},
	{[]string{"is not", "does not equal"}, "Different", "não é, é diferente de", ComparatorCategory},
	{[]string{"is greater than"}, "Strictly greater", "é maior que", ComparatorCategory},
	{[]string{"is less than"}, "Strictly less", "é menor que", ComparatorCategory},
	{[]string{"is at least"}, "Greater or equal", "é no mínimo", ComparatorCategory},
	{[]string{"is at most"}, "Less or equal", "é no máximo", ComparatorCategory},
	{
		[]string{"is between"},
		"Greater or equal to X and less or equal to Y (both inclusive)",
		"está entre X e Y, inclusive",
		ComparatorCategory,
	},
	{
		[]string{"is strictly between"},
		"Greater than X and less than Y",
		"está estritamente entre X e Y",
		ComparatorCategory,
	},
	{[]string{"is one of"}, "Belongs to the list", "é um de", ComparatorCategory},
	{
		[]string{"contains", "does not contain"},
		"The collection or text contains (or does not contain) the element",
		"contém, não contém",
		ComparatorCategory,
	},
	{
		[]string{"is empty", "is not empty"},
		"Collection or text with no elements (or with some)",
		"está vazio, não está vazio",
		ComparatorCategory,
	},
	{
		[]string{"every item has", "at least one item has", "no item has"},
		"Quantifiers over collections",
		"todo item tem, pelo menos um item tem, nenhum item tem",
		QuantifierCategory,
	},
	{
		[]string{"remains", "does not change"},
		"Equal to the value from another execution of the relation",
		"continua, não muda",
		ComparatorCategory,
	},
	{
		[]string{"is before", "is after", "is within N days of"},
		"Date comparisons",
		"é antes de, é depois de, está a no máximo N dias de",
		DateCategory,
	},
	{
		[]string{"plus", "minus", "times", "divided by", "percent of"},
		"Exact-decimal arithmetic",
		"mais, menos, vezes, dividido por, por cento de",
		ArithmeticCategory,
	},
	{
		[]string{"rounded to N places"},
		"Declared rounding (mandatory after \"divided by\" and \"percent of\")",
		"arredondado a N casas",
		ArithmeticCategory,
	},
	{
		[]string{"and", "or", "not"},
		"Logic. Mixing \"and\" and \"or\" in the same statement is refused " +
			"(there are no parentheses): use separate statements or a decision table instead",
		"e, ou, não",
		LogicCategory,
	},
	{
		[]string{
			"an item is added", "an item is removed", "the items are reordered",
			"the operation is repeated", "the operation is undone",
		},
		"Catalog transformations, used in \"keeps\": monotonicity, permutation, idempotency, operation and undo",
		"um item é adicionado, um item é removido, os itens são reordenados, a operação é repetida, a operação é desfeita",
		CatalogCategory,
	},
	{
		[]string{"N ms", "N s", "N min"},
		"Duration, checked against the concept's declared unit",
		"N milissegundos, N segundos, N minutos",
		DurationCategory,
	},
	{
		[]string{"in at least N of M calls"},
		"Frequency: the statement holds in at least N of M observed calls",
		"em pelo menos N de M chamadas",
		FrequencyCategory,
	},
}

// vocabularyIndex maps every known phrase to its entry, built once from
// [Vocabulary].
var vocabularyIndex = buildVocabularyIndex()

func buildVocabularyIndex() map[string]*VocabularyEntry {
	index := make(map[string]*VocabularyEntry)
	for i := range Vocabulary {
		entry := &Vocabulary[i]
		for _, phrase := range entry.Phrases {
			if existing, dup := index[phrase]; dup {
				panic(fmt.Sprintf("cjtr: vocabulary phrase %q is declared twice (categories %q and %q)",
					phrase, existing.Category, entry.Category))
			}
			index[phrase] = entry
		}
	}
	return index
}

// IsKnownPhrase reports whether phrase is exactly one of the controlled
// vocabulary's fixed phrases (case-sensitive: the vocabulary is declared
// in lowercase English, same as every worked example).
//
// This does not yet recognize a phrase's parameterized forms (e.g. "is
// within N days of" matching "is within 3 days of") or try to parse a full
// statement - it only checks membership in the fixed phrase set itself.
func IsKnownPhrase(phrase string) bool {
	_, ok := vocabularyIndex[phrase]
	return ok
}

// LookupPhrase returns the vocabulary entry a phrase belongs to, if any.
func LookupPhrase(phrase string) (VocabularyEntry, bool) {
	entry, ok := vocabularyIndex[phrase]
	if !ok {
		return VocabularyEntry{}, false
	}
	return *entry, true
}
