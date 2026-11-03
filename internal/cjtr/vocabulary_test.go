package cjtr

import "testing"

func TestVocabulary_NoDuplicatePhrases(t *testing.T) {
	seen := make(map[string]VocabularyCategory)
	for _, entry := range Vocabulary {
		for _, phrase := range entry.Phrases {
			if cat, dup := seen[phrase]; dup {
				t.Errorf("phrase %q appears in both %q and %q", phrase, cat, entry.Category)
			}
			seen[phrase] = entry.Category
		}
	}
}

func TestVocabulary_NoEmptyEntries(t *testing.T) {
	for _, entry := range Vocabulary {
		if len(entry.Phrases) == 0 {
			t.Errorf("entry with meaning %q has no phrases", entry.Meaning)
		}
		if entry.Meaning == "" {
			t.Errorf("entry with phrases %v has no meaning", entry.Phrases)
		}
		if entry.PortugueseReading == "" {
			t.Errorf("entry with phrases %v has no Portuguese reading", entry.Phrases)
		}
		for _, phrase := range entry.Phrases {
			if phrase == "" {
				t.Errorf("entry with meaning %q has an empty phrase", entry.Meaning)
			}
		}
	}
}

func TestIsKnownPhrase(t *testing.T) {
	tests := []struct {
		phrase string
		want   bool
	}{
		{"is", true},
		{"is greater than", true},
		{"is between", true},
		{"rounded to N places", true},
		{"and", true},
		{"is somewhat like", false},
		{"", false},
		{"Is", false}, // case-sensitive: the vocabulary is lowercase
	}

	for _, tt := range tests {
		if got := IsKnownPhrase(tt.phrase); got != tt.want {
			t.Errorf("IsKnownPhrase(%q) = %v, want %v", tt.phrase, got, tt.want)
		}
	}
}

func TestLookupPhrase(t *testing.T) {
	entry, ok := LookupPhrase("is greater than")
	if !ok {
		t.Fatal("LookupPhrase(\"is greater than\") not found")
	}
	if entry.Category != ComparatorCategory {
		t.Errorf("category = %q, want %q", entry.Category, ComparatorCategory)
	}
	if entry.Meaning != "Strictly greater" {
		t.Errorf("meaning = %q, want %q", entry.Meaning, "Strictly greater")
	}

	// Synonyms share one entry.
	isEntry, _ := LookupPhrase("is")
	equalsEntry, _ := LookupPhrase("equals")
	if isEntry.Meaning != equalsEntry.Meaning {
		t.Errorf("\"is\" and \"equals\" should share one entry, got meanings %q and %q", isEntry.Meaning, equalsEntry.Meaning)
	}

	_, ok = LookupPhrase("not a real phrase")
	if ok {
		t.Error("LookupPhrase(\"not a real phrase\") found an entry, want none")
	}
}
