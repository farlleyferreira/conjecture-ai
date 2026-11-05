package cjtr

import (
	"fmt"
	"strings"
	"testing"
)

const unsignedDoc = `conjecture: 1.0.0

phenomenon: free shipping above the threshold

concepts:
  shipping fee:
    variable: output.shipping_fee
    type: decimal
`

func TestFindPendingMarkers_None(t *testing.T) {
	markers := FindPendingMarkers([]byte(unsignedDoc))
	if len(markers) != 0 {
		t.Errorf("FindPendingMarkers = %v, want none", markers)
	}
}

func TestFindPendingMarkers_Found(t *testing.T) {
	src := "then: ???\notherwise: the shipping fee is ???\n"

	markers := FindPendingMarkers([]byte(src))
	if len(markers) != 2 {
		t.Fatalf("len(FindPendingMarkers) = %d, want 2", len(markers))
	}
	if markers[0].Line != 1 || markers[0].Text != "then: ???" {
		t.Errorf("markers[0] = %+v", markers[0])
	}
	if markers[1].Line != 2 || markers[1].Text != "otherwise: the shipping fee is ???" {
		t.Errorf("markers[1] = %+v", markers[1])
	}
}

func TestCheckSignable_NoMarkers(t *testing.T) {
	if err := CheckSignable([]byte(unsignedDoc)); err != nil {
		t.Errorf("CheckSignable returned unexpected error: %v", err)
	}
}

func TestCheckSignable_BlocksOnMarkers(t *testing.T) {
	err := CheckSignable([]byte("then: ???\n"))
	if err == nil {
		t.Fatal("CheckSignable with an unresolved \"???\" marker = nil error, want an error")
	}
	if !strings.Contains(err.Error(), "line 1") {
		t.Errorf("error = %q, want it to name the marker's line", err.Error())
	}
}

func TestReadSigned_NoBlock(t *testing.T) {
	root := loadNode(t, unsignedDoc)

	signed, err := ReadSigned(root)
	if err != nil {
		t.Fatalf("ReadSigned returned unexpected error: %v", err)
	}
	if signed != nil {
		t.Errorf("ReadSigned with no \"signed\" key = %+v, want nil", signed)
	}
}

func TestReadSigned_FullBlock(t *testing.T) {
	src := unsignedDoc + `
signed:
  user: alice
  tool: conjecture-cli
  date: "2026-11-05"
  hash: deadbeef
  confirmed: true
`
	root := loadNode(t, src)

	signed, err := ReadSigned(root)
	if err != nil {
		t.Fatalf("ReadSigned returned unexpected error: %v", err)
	}
	want := &Signed{User: "alice", Tool: "conjecture-cli", Date: "2026-11-05", Hash: "deadbeef", Confirmed: true}
	if *signed != *want {
		t.Errorf("ReadSigned = %+v, want %+v", signed, want)
	}
}

func TestReadSigned_InvalidConfirmed(t *testing.T) {
	src := unsignedDoc + "\nsigned:\n  confirmed: yes\n"
	root := loadNode(t, src)

	_, err := ReadSigned(root)
	if err == nil {
		t.Fatal("ReadSigned with \"confirmed: yes\" (not true/false) = nil error, want an error")
	}
}

func TestContentHash_StableWithNoSignedBlockYet(t *testing.T) {
	root := loadNode(t, unsignedDoc)

	h1, err := ContentHash([]byte(unsignedDoc), root)
	if err != nil {
		t.Fatalf("ContentHash returned unexpected error: %v", err)
	}
	h2, err := ContentHash([]byte(unsignedDoc), root)
	if err != nil {
		t.Fatalf("ContentHash returned unexpected error: %v", err)
	}
	if fmt.Sprintf("%x", h1) != fmt.Sprintf("%x", h2) {
		t.Error("ContentHash gave different results for identical input")
	}
}

func TestContentHash_SigningLaterReproducesTheSameHash(t *testing.T) {
	// Signing should not retroactively change the hash of what came
	// before the "signed" block: computing the hash before signing, and
	// computing it again afterward (with "signed" now present and
	// excluded), must agree.
	before := unsignedDoc
	rootBefore := loadNode(t, before)
	hashBefore, err := ContentHash([]byte(before), rootBefore)
	if err != nil {
		t.Fatalf("ContentHash returned unexpected error: %v", err)
	}

	after := before + fmt.Sprintf("signed:\n  user: alice\n  hash: %x\n", hashBefore)
	rootAfter := loadNode(t, after)
	hashAfter, err := ContentHash([]byte(after), rootAfter)
	if err != nil {
		t.Fatalf("ContentHash returned unexpected error: %v", err)
	}

	if fmt.Sprintf("%x", hashBefore) != fmt.Sprintf("%x", hashAfter) {
		t.Error("ContentHash changed after appending a \"signed\" block, want it unchanged")
	}
}

func TestContentHash_ChangeOutsideSignedBlockInvalidates(t *testing.T) {
	signed := unsignedDoc + "\nsigned:\n  user: alice\n  hash: irrelevant\n"
	root := loadNode(t, signed)
	hash, err := ContentHash([]byte(signed), root)
	if err != nil {
		t.Fatalf("ContentHash returned unexpected error: %v", err)
	}

	// A change outside the "signed" block - here, a comment, which this
	// package's own loader would not even look at - must still change
	// the hash.
	changed := strings.Replace(signed, "phenomenon: free shipping above the threshold",
		"phenomenon: free shipping above the threshold # a harmless comment", 1)
	changedRoot := loadNode(t, changed)
	changedHash, err := ContentHash([]byte(changed), changedRoot)
	if err != nil {
		t.Fatalf("ContentHash returned unexpected error: %v", err)
	}

	if fmt.Sprintf("%x", hash) == fmt.Sprintf("%x", changedHash) {
		t.Error("ContentHash did not change after a change outside the \"signed\" block")
	}
}

func TestContentHash_ChangeInsideSignedBlockDoesNotAffectHash(t *testing.T) {
	// The hash is computed to be put INTO the signed block, so by
	// definition changing values already inside that block (e.g.
	// re-signing with a different user) must not itself change the
	// computed hash of everything else.
	a := unsignedDoc + "\nsigned:\n  user: alice\n  hash: irrelevant\n"
	b := unsignedDoc + "\nsigned:\n  user: bob\n  hash: also-irrelevant\n"

	rootA := loadNode(t, a)
	hashA, err := ContentHash([]byte(a), rootA)
	if err != nil {
		t.Fatalf("ContentHash returned unexpected error: %v", err)
	}
	rootB := loadNode(t, b)
	hashB, err := ContentHash([]byte(b), rootB)
	if err != nil {
		t.Fatalf("ContentHash returned unexpected error: %v", err)
	}

	if fmt.Sprintf("%x", hashA) != fmt.Sprintf("%x", hashB) {
		t.Error("ContentHash differed between two documents that differ only inside the \"signed\" block")
	}
}

func TestVerifySignature_Valid(t *testing.T) {
	root := loadNode(t, unsignedDoc)
	hash, err := ContentHash([]byte(unsignedDoc), root)
	if err != nil {
		t.Fatalf("ContentHash returned unexpected error: %v", err)
	}

	signed := unsignedDoc + fmt.Sprintf("signed:\n  user: alice\n  hash: %x\n", hash)
	signedRoot := loadNode(t, signed)

	ok, err := VerifySignature([]byte(signed), signedRoot)
	if err != nil {
		t.Fatalf("VerifySignature returned unexpected error: %v", err)
	}
	if !ok {
		t.Error("VerifySignature = false, want true (the recorded hash matches)")
	}
}

func TestVerifySignature_InvalidatedByChange(t *testing.T) {
	root := loadNode(t, unsignedDoc)
	hash, err := ContentHash([]byte(unsignedDoc), root)
	if err != nil {
		t.Fatalf("ContentHash returned unexpected error: %v", err)
	}

	signed := unsignedDoc + fmt.Sprintf("signed:\n  user: alice\n  hash: %x\n", hash)
	tampered := strings.Replace(signed, "free shipping above the threshold", "something else entirely", 1)
	tamperedRoot := loadNode(t, tampered)

	ok, err := VerifySignature([]byte(tampered), tamperedRoot)
	if err != nil {
		t.Fatalf("VerifySignature returned unexpected error: %v", err)
	}
	if ok {
		t.Error("VerifySignature = true after a change outside \"signed\", want false")
	}
}

func TestVerifySignature_NoSignedBlock(t *testing.T) {
	root := loadNode(t, unsignedDoc)

	_, err := VerifySignature([]byte(unsignedDoc), root)
	if err == nil {
		t.Fatal("VerifySignature on an unsigned file = nil error, want an error")
	}
}
