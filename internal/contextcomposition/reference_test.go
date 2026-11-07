package contextcomposition

import "testing"

func TestResolveReference_FollowsLatestWhenSeqIsZero(t *testing.T) {
	db := openTestStore(t)
	if _, err := CreateContext(db, "source"); err != nil {
		t.Fatalf("CreateContext returned unexpected error: %v", err)
	}
	if _, err := AddVersion(db, "source", "v1 content"); err != nil {
		t.Fatalf("AddVersion returned unexpected error: %v", err)
	}

	got, err := ResolveReference(db, Reference{Name: "source"})
	if err != nil {
		t.Fatalf("ResolveReference returned unexpected error: %v", err)
	}
	if got.Content != "v1 content" {
		t.Errorf("ResolveReference content = %q, want %q", got.Content, "v1 content")
	}
}

func TestResolveReference_DoesNotCopyContent_SeesLaterVersionsOfTheSameSource(t *testing.T) {
	// "Composition by reference does not copy content": the composed
	// context points to the source's content rather than duplicating
	// it. Proven here by resolving the same zero-Seq Reference twice,
	// with a new version added to the source in between - a copy taken
	// once would never see it.
	db := openTestStore(t)
	if _, err := CreateContext(db, "source"); err != nil {
		t.Fatalf("CreateContext returned unexpected error: %v", err)
	}
	if _, err := AddVersion(db, "source", "v1 content"); err != nil {
		t.Fatalf("AddVersion returned unexpected error: %v", err)
	}

	ref := Reference{Name: "source"}
	first, err := ResolveReference(db, ref)
	if err != nil {
		t.Fatalf("ResolveReference returned unexpected error: %v", err)
	}
	if first.Content != "v1 content" {
		t.Fatalf("first resolve content = %q, want %q", first.Content, "v1 content")
	}

	if _, err := AddVersion(db, "source", "v2 content"); err != nil {
		t.Fatalf("AddVersion returned unexpected error: %v", err)
	}

	second, err := ResolveReference(db, ref)
	if err != nil {
		t.Fatalf("ResolveReference returned unexpected error: %v", err)
	}
	if second.Content != "v2 content" {
		t.Errorf("second resolve content = %q, want %q (the same Reference must track the source's current content, not a copy)", second.Content, "v2 content")
	}
}

func TestResolveReference_PinnedSeqIgnoresLaterVersions(t *testing.T) {
	db := openTestStore(t)
	if _, err := CreateContext(db, "source"); err != nil {
		t.Fatalf("CreateContext returned unexpected error: %v", err)
	}
	if _, err := AddVersion(db, "source", "v1 content"); err != nil {
		t.Fatalf("AddVersion returned unexpected error: %v", err)
	}
	if _, err := AddVersion(db, "source", "v2 content"); err != nil {
		t.Fatalf("AddVersion returned unexpected error: %v", err)
	}

	got, err := ResolveReference(db, Reference{Name: "source", Seq: 1})
	if err != nil {
		t.Fatalf("ResolveReference returned unexpected error: %v", err)
	}
	if got.Content != "v1 content" {
		t.Errorf("ResolveReference(Seq: 1) content = %q, want %q", got.Content, "v1 content")
	}
}

func TestResolveReference_UnknownSourceIsError(t *testing.T) {
	db := openTestStore(t)
	_, err := ResolveReference(db, Reference{Name: "does-not-exist"})
	if err == nil {
		t.Fatal("ResolveReference for an unregistered context = nil error, want one")
	}
}
