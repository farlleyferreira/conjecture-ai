package languageagnosticcontracts

import (
	"context"
	"errors"
	"testing"
)

// errBoom is a shared stand-in failure, reused by both platform
// backends' own test files (guidriver_windows_test.go,
// guidriver_linux_test.go) - defined here, in a file with no GOOS
// suffix, since only one of those two ever compiles for a given
// target.
var errBoom = errors.New("boom")

func TestRole_String(t *testing.T) {
	tests := []struct {
		r    Role
		want string
	}{
		{Window, "window"},
		{Button, "button"},
		{TextField, "text field"},
		{Label, "label"},
		{CheckBox, "check box"},
		{MenuItem, "menu item"},
		{UnknownRole, "unknown"},
	}
	for _, tt := range tests {
		if got := tt.r.String(); got != tt.want {
			t.Errorf("Role(%d).String() = %q, want %q", int(tt.r), got, tt.want)
		}
	}
}

func TestFindByName_FindsTheElementItself(t *testing.T) {
	root := GUIElement{Role: Window, Name: "Main Window"}
	got, found := FindByName(root, "Main Window")
	if !found {
		t.Fatal("FindByName found = false, want true (the root itself matches)")
	}
	if got.Role != Window {
		t.Errorf("got.Role = %v, want Window", got.Role)
	}
}

func TestFindByName_FindsANestedDescendant(t *testing.T) {
	root := GUIElement{
		Role: Window, Name: "Main Window",
		Children: []GUIElement{
			{Role: Label, Name: "Status"},
			{
				Role: TextField, Name: "Toolbar",
				Children: []GUIElement{
					{Role: Button, Name: "Submit", Value: ""},
				},
			},
		},
	}

	got, found := FindByName(root, "Submit")
	if !found {
		t.Fatal("FindByName found = false, want true (nested two levels deep)")
	}
	if got.Role != Button {
		t.Errorf("got.Role = %v, want Button", got.Role)
	}
}

func TestFindByName_NotFoundReportsFalse(t *testing.T) {
	root := GUIElement{Role: Window, Name: "Main Window", Children: []GUIElement{{Role: Label, Name: "Status"}}}

	_, found := FindByName(root, "Nonexistent")
	if found {
		t.Error("FindByName found = true for a name not present in the tree, want false")
	}
}

// fakeDriver is a minimal [Driver] implementation, confirming the
// interface is actually usable by a caller holding one generically -
// no real per-OS backend exists yet (see guidriver.go's own doc
// comment), so this is the only thing exercising Driver as an
// interface until one does.
type fakeDriver struct {
	snapshot GUIElement
	err      error
}

func (f *fakeDriver) Snapshot(context.Context) (GUIElement, error) {
	return f.snapshot, f.err
}

func TestDriver_InterfaceIsUsableGenerically(t *testing.T) {
	var d Driver = &fakeDriver{snapshot: GUIElement{Role: Window, Name: "Main Window"}}

	got, err := d.Snapshot(context.Background())
	if err != nil {
		t.Fatalf("Snapshot returned unexpected error: %v", err)
	}
	if got.Name != "Main Window" {
		t.Errorf("got.Name = %q, want %q", got.Name, "Main Window")
	}
}
