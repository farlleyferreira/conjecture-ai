package languageagnosticcontracts

import (
	"context"
	"os"
	"testing"
)

func TestParseWindowsSnapshot_SimpleLeaf(t *testing.T) {
	got, err := parseWindowsSnapshot([]byte(`{"Role":"ControlType.Window","Name":"Notepad","Value":"","Children":[]}`))
	if err != nil {
		t.Fatalf("parseWindowsSnapshot returned unexpected error: %v", err)
	}
	want := GUIElement{Role: Window, Name: "Notepad"}
	if got.Role != want.Role || got.Name != want.Name || len(got.Children) != 0 {
		t.Errorf("parseWindowsSnapshot = %+v, want %+v", got, want)
	}
}

func TestParseWindowsSnapshot_NestedChildrenArray(t *testing.T) {
	data := `{"Role":"ControlType.Window","Name":"Calc","Value":"","Children":[
		{"Role":"ControlType.Edit","Name":"Result","Value":"42","Children":[]},
		{"Role":"ControlType.Button","Name":"Equals","Value":"","Children":[]}
	]}`
	got, err := parseWindowsSnapshot([]byte(data))
	if err != nil {
		t.Fatalf("parseWindowsSnapshot returned unexpected error: %v", err)
	}
	if len(got.Children) != 2 {
		t.Fatalf("got %d children, want 2", len(got.Children))
	}
	if got.Children[0].Role != TextField || got.Children[0].Value != "42" {
		t.Errorf("first child = %+v, want a TextField with value 42", got.Children[0])
	}
	if got.Children[1].Role != Button || got.Children[1].Name != "Equals" {
		t.Errorf("second child = %+v, want a Button named Equals", got.Children[1])
	}
}

// TestParseWindowsSnapshot_SingleChildCollapsedToObject guards against
// windowsSnapshotScript's own doc-noted PowerShell quirk: a
// one-element Children array rendered as a bare JSON object rather
// than a one-element array.
func TestParseWindowsSnapshot_SingleChildCollapsedToObject(t *testing.T) {
	data := `{"Role":"ControlType.Window","Name":"Dialog","Value":"","Children":{"Role":"ControlType.Button","Name":"OK","Value":"","Children":[]}}`
	got, err := parseWindowsSnapshot([]byte(data))
	if err != nil {
		t.Fatalf("parseWindowsSnapshot returned unexpected error: %v", err)
	}
	if len(got.Children) != 1 || got.Children[0].Name != "OK" {
		t.Errorf("parseWindowsSnapshot = %+v, want one child named OK", got)
	}
}

func TestParseWindowsSnapshot_NullChildrenIsNoChildren(t *testing.T) {
	got, err := parseWindowsSnapshot([]byte(`{"Role":"ControlType.Text","Name":"Label","Value":"","Children":null}`))
	if err != nil {
		t.Fatalf("parseWindowsSnapshot returned unexpected error: %v", err)
	}
	if got.Children != nil {
		t.Errorf("Children = %+v, want nil", got.Children)
	}
}

func TestParseWindowsSnapshot_UnrecognizedControlTypeIsUnknownRole(t *testing.T) {
	got, err := parseWindowsSnapshot([]byte(`{"Role":"ControlType.Slider","Name":"Volume","Value":"","Children":[]}`))
	if err != nil {
		t.Fatalf("parseWindowsSnapshot returned unexpected error: %v", err)
	}
	if got.Role != UnknownRole {
		t.Errorf("Role = %v, want UnknownRole", got.Role)
	}
}

func TestParseWindowsSnapshot_MalformedJSONIsError(t *testing.T) {
	if _, err := parseWindowsSnapshot([]byte("not json")); err == nil {
		t.Error("parseWindowsSnapshot on malformed JSON = nil error, want one")
	}
}

func TestWindowsDriver_Snapshot_UsesRunHookAndParsesResult(t *testing.T) {
	d := &WindowsDriver{
		ProcessName: "notepad.exe",
		run: func(ctx context.Context, script string) ([]byte, error) {
			if script == "" {
				t.Error("script passed to run is empty")
			}
			return []byte(`{"Role":"ControlType.Window","Name":"Notepad","Value":"","Children":[]}`), nil
		},
	}
	got, err := d.Snapshot(context.Background())
	if err != nil {
		t.Fatalf("Snapshot returned unexpected error: %v", err)
	}
	if got.Role != Window || got.Name != "Notepad" {
		t.Errorf("Snapshot = %+v, want the parsed root window", got)
	}
}

func TestWindowsDriver_Snapshot_RunErrorIsWrapped(t *testing.T) {
	d := &WindowsDriver{
		run: func(ctx context.Context, script string) ([]byte, error) {
			return nil, errBoom
		},
	}
	_, err := d.Snapshot(context.Background())
	if err == nil {
		t.Fatal("Snapshot with a failing run hook = nil error, want one")
	}
}

// TestWindowsDriver_RealTarget is a real, manual integration test
// against an actually running Notepad instance and this machine's own
// real desktop session - never run in CI (ubuntu-latest never
// compiles this Windows-only file in the first place, so this guard
// is for a future Windows CI runner, or a human running this suite
// directly on Windows, not for this project's current pipeline).
// Start "notepad.exe" by hand, set CONJECTURE_UIA_MANUAL_TEST=1, then
// run this test alone to verify the real backend against a real
// window.
func TestWindowsDriver_RealTarget(t *testing.T) {
	if os.Getenv("CONJECTURE_UIA_MANUAL_TEST") == "" {
		t.Skip("manual only - set CONJECTURE_UIA_MANUAL_TEST=1 with notepad.exe already running")
	}
	d := &WindowsDriver{ProcessName: "notepad.exe"}
	root, err := d.Snapshot(context.Background())
	if err != nil {
		t.Fatalf("Snapshot against the real notepad.exe returned unexpected error: %v", err)
	}
	if root.Role != Window {
		t.Errorf("root.Role = %v, want Window", root.Role)
	}
	if len(root.Children) == 0 {
		t.Error("root has no children - expected a real Notepad window's own accessibility tree")
	}
	// Checked by Role, not Name: Notepad's own accessible names are
	// localized (this machine's own real run surfaced "Arquivo"/
	// "Editar"/"Exibir", not "File"/"Edit"/"View"), but a real menu
	// bar's MenuItem role is the same regardless of locale.
	if countRole(root, MenuItem) == 0 {
		t.Error("found no MenuItem anywhere in the real tree - expected Notepad's own menu bar")
	}
	if countRole(root, Button) == 0 {
		t.Error("found no Button anywhere in the real tree - expected Notepad's own toolbar buttons")
	}
}

// countRole counts every element in root's own tree (itself and every
// descendant) whose Role is role.
func countRole(root GUIElement, role Role) int {
	count := 0
	if root.Role == role {
		count++
	}
	for _, child := range root.Children {
		count += countRole(child, role)
	}
	return count
}
