package languageagnosticcontracts

import (
	"os"
	"testing"
)

// fakeAtspiAccessible is a hand-built [atspiAccessible] for testing
// [atspiSnapshot]/[selectApp]/[topLevelWindow] with no real D-Bus
// connection or desktop session involved.
type fakeAtspiAccessible struct {
	name     string
	role     string
	text     string
	children []atspiAccessible
	err      error
}

func (f *fakeAtspiAccessible) Name() (string, error)     { return f.name, f.err }
func (f *fakeAtspiAccessible) RoleName() (string, error) { return f.role, f.err }
func (f *fakeAtspiAccessible) Text() (string, error)     { return f.text, f.err }
func (f *fakeAtspiAccessible) Children() ([]atspiAccessible, error) {
	return f.children, f.err
}

func TestAtspiSnapshot_SimpleLeaf(t *testing.T) {
	root := &fakeAtspiAccessible{name: "MyApp", role: "frame"}
	got, err := atspiSnapshot(root)
	if err != nil {
		t.Fatalf("atspiSnapshot returned unexpected error: %v", err)
	}
	if got.Role != Window || got.Name != "MyApp" {
		t.Errorf("atspiSnapshot = %+v, want a Window named MyApp", got)
	}
}

func TestAtspiSnapshot_NestedChildrenAndText(t *testing.T) {
	root := &fakeAtspiAccessible{
		name: "Window", role: "frame",
		children: []atspiAccessible{
			&fakeAtspiAccessible{name: "Username", role: "entry", text: "alice"},
			&fakeAtspiAccessible{name: "Submit", role: "push button"},
		},
	}
	got, err := atspiSnapshot(root)
	if err != nil {
		t.Fatalf("atspiSnapshot returned unexpected error: %v", err)
	}
	if len(got.Children) != 2 {
		t.Fatalf("got %d children, want 2", len(got.Children))
	}
	if got.Children[0].Role != TextField || got.Children[0].Value != "alice" {
		t.Errorf("first child = %+v, want a TextField with value \"alice\"", got.Children[0])
	}
	if got.Children[1].Role != Button || got.Children[1].Name != "Submit" {
		t.Errorf("second child = %+v, want a Button named Submit", got.Children[1])
	}
}

func TestAtspiSnapshot_UnrecognizedRoleIsUnknownRole(t *testing.T) {
	root := &fakeAtspiAccessible{name: "Thing", role: "slider"}
	got, err := atspiSnapshot(root)
	if err != nil {
		t.Fatalf("atspiSnapshot returned unexpected error: %v", err)
	}
	if got.Role != UnknownRole {
		t.Errorf("Role = %v, want UnknownRole", got.Role)
	}
}

func TestAtspiSnapshot_ErrorPropagates(t *testing.T) {
	root := &fakeAtspiAccessible{err: errBoom}
	if _, err := atspiSnapshot(root); err == nil {
		t.Fatal("atspiSnapshot with a failing accessible = nil error, want one")
	}
}

func TestSelectApp_EmptyNameWithOneAppPicksIt(t *testing.T) {
	app := &fakeAtspiAccessible{name: "OnlyApp"}
	got, err := selectApp([]atspiAccessible{app}, "")
	if err != nil {
		t.Fatalf("selectApp returned unexpected error: %v", err)
	}
	if got != app {
		t.Error("selectApp did not return the single registered application")
	}
}

func TestSelectApp_EmptyNameWithMultipleAppsIsError(t *testing.T) {
	apps := []atspiAccessible{&fakeAtspiAccessible{name: "A"}, &fakeAtspiAccessible{name: "B"}}
	if _, err := selectApp(apps, ""); err == nil {
		t.Fatal("selectApp with an empty name and multiple apps = nil error, want one")
	}
}

func TestSelectApp_MatchesCaseInsensitiveSubstring(t *testing.T) {
	apps := []atspiAccessible{
		&fakeAtspiAccessible{name: "Text Editor"},
		&fakeAtspiAccessible{name: "Calculator"},
	}
	got, err := selectApp(apps, "CALC")
	if err != nil {
		t.Fatalf("selectApp returned unexpected error: %v", err)
	}
	if got != apps[1] {
		t.Error("selectApp did not match \"CALC\" against \"Calculator\"")
	}
}

func TestSelectApp_NoMatchIsError(t *testing.T) {
	apps := []atspiAccessible{&fakeAtspiAccessible{name: "A"}}
	if _, err := selectApp(apps, "nonexistent"); err == nil {
		t.Fatal("selectApp with no matching app = nil error, want one")
	}
}

func TestTopLevelWindow_FindsFrameChild(t *testing.T) {
	frame := &fakeAtspiAccessible{name: "Main Window", role: "frame"}
	app := &fakeAtspiAccessible{name: "App", role: "application", children: []atspiAccessible{frame}}
	got, err := topLevelWindow(app)
	if err != nil {
		t.Fatalf("topLevelWindow returned unexpected error: %v", err)
	}
	if got != frame {
		t.Error("topLevelWindow did not return the frame child")
	}
}

func TestTopLevelWindow_FallsBackToAppWhenNoFrame(t *testing.T) {
	app := &fakeAtspiAccessible{name: "App", role: "application"}
	got, err := topLevelWindow(app)
	if err != nil {
		t.Fatalf("topLevelWindow returned unexpected error: %v", err)
	}
	if got != app {
		t.Error("topLevelWindow did not fall back to the application object")
	}
}

// TestLinuxDriver_RealTarget is a real, manual integration test
// against an actual running GTK application and this project's own
// real AT-SPI accessibility bus (Xvfb + dbus + at-spi2-core) - never
// run in CI (this project's own ubuntu-latest CI runner has no
// desktop session or accessibility bus running). Set
// CONJECTURE_ATSPI_MANUAL_TEST to the target application's own
// accessible name (or a substring of it) before running this test
// alone.
func TestLinuxDriver_RealTarget(t *testing.T) {
	appName, isSet := os.LookupEnv("CONJECTURE_ATSPI_MANUAL_TEST")
	if !isSet {
		t.Skip("manual only - set CONJECTURE_ATSPI_MANUAL_TEST=<app name> (or \"\" if exactly one app is registered) with a real GTK app already running under a real AT-SPI bus")
	}
	d := &LinuxDriver{AppName: appName}
	root, err := d.Snapshot(t.Context())
	if err != nil {
		t.Fatalf("Snapshot against the real AT-SPI bus returned unexpected error: %v", err)
	}
	if root.Role != Window {
		t.Errorf("root.Role = %v, want Window", root.Role)
	}
	if len(root.Children) == 0 {
		t.Error("root has no children - expected a real application window's own accessibility tree")
	}
}
