package omissiondetection

import "testing"

func TestCheckTypeD_CoveredCategoryNotFlagged(t *testing.T) {
	catalog := Catalog{"concurrency"}
	addressed := map[string]bool{"concurrency": true}

	got := CheckTypeD(catalog, "shipping", addressed)
	if len(got) != 0 {
		t.Errorf("CheckTypeD = %v, want no omissions (the only catalog category is addressed)", got)
	}
}

func TestCheckTypeD_UncoveredCategoryFlagged(t *testing.T) {
	catalog := Catalog{"concurrency", "localization", "accessibility"}
	addressed := map[string]bool{"concurrency": true}

	got := CheckTypeD(catalog, "shipping", addressed)
	want := []TypeDOmission{
		{Area: "shipping", Category: "localization"},
		{Area: "shipping", Category: "accessibility"},
	}
	if len(got) != len(want) {
		t.Fatalf("CheckTypeD returned %d omissions, want %d: got %v", len(got), len(want), got)
	}
	for i, w := range want {
		if got[i] != w {
			t.Errorf("got[%d] = %+v, want %+v", i, got[i], w)
		}
	}
}

func TestCheckTypeD_CategoryOutsideCatalogNeverAppears(t *testing.T) {
	// "A category outside this catalog cannot be detected": even
	// though "security" is clearly unaddressed, it was never in the
	// catalog, so CheckTypeD must never mention it.
	catalog := Catalog{"concurrency"}
	addressed := map[string]bool{} // nothing addressed, including "security"

	got := CheckTypeD(catalog, "shipping", addressed)
	for _, o := range got {
		if o.Category == "security" {
			t.Fatal("CheckTypeD reported \"security\", a category outside the configured catalog")
		}
	}
	if len(got) != 1 || got[0].Category != "concurrency" {
		t.Errorf("CheckTypeD = %v, want exactly [concurrency]", got)
	}
}

func TestCheckTypeD_EmptyCatalogNeverFlagsAnything(t *testing.T) {
	// An empty result here must not be read as "no omissions exist" -
	// it only reflects an empty catalog, which checked nothing at all.
	got := CheckTypeD(nil, "shipping", map[string]bool{})
	if len(got) != 0 {
		t.Errorf("CheckTypeD(nil catalog) = %v, want none", got)
	}
}
