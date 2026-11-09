package teamconfig

import (
	"bytes"
	"testing"

	"github.com/farlleyferreira/conjecture-ai/internal/yamlstrict"
)

func loadRoot(t *testing.T, content string) *yamlstrict.Node {
	t.Helper()
	root, err := yamlstrict.Load(bytes.NewReader([]byte(content)))
	if err != nil {
		t.Fatalf("yamlstrict.Load returned unexpected error: %v", err)
	}
	return root
}

func TestLoadSettings_NoSettingsKeyReturnsNil(t *testing.T) {
	root := loadRoot(t, `conjecture: 1.0.0`)

	settings, err := LoadSettings(root)
	if err != nil {
		t.Fatalf("LoadSettings returned unexpected error: %v", err)
	}
	if settings != nil {
		t.Errorf("settings = %v, want nil (no \"settings\" key in the file)", settings)
	}
}

func TestLoadSettings_ParsesEveryEntry(t *testing.T) {
	root := loadRoot(t, `
conjecture: 1.0.0

settings:
  law_contested_threshold:
    floor: 3
    direction: lower
  queue_aging_rate:
    floor: 1.0
    direction: higher
`)

	settings, err := LoadSettings(root)
	if err != nil {
		t.Fatalf("LoadSettings returned unexpected error: %v", err)
	}
	if len(settings) != 2 {
		t.Fatalf("len(settings) = %d, want 2", len(settings))
	}
	got, ok := settings["law_contested_threshold"]
	if !ok {
		t.Fatal("settings is missing \"law_contested_threshold\"")
	}
	if got.Floor != 3 || got.Direction != Lower {
		t.Errorf("law_contested_threshold = %+v, want {Floor: 3, Direction: Lower}", got)
	}
	got, ok = settings["queue_aging_rate"]
	if !ok {
		t.Fatal("settings is missing \"queue_aging_rate\"")
	}
	if got.Floor != 1.0 || got.Direction != Higher {
		t.Errorf("queue_aging_rate = %+v, want {Floor: 1.0, Direction: Higher}", got)
	}
}

func TestLoadSettings_NotAMappingIsError(t *testing.T) {
	root := loadRoot(t, `
conjecture: 1.0.0
settings: not-a-mapping
`)

	_, err := LoadSettings(root)
	if err == nil {
		t.Fatal("LoadSettings with a non-mapping \"settings\" = nil error, want one")
	}
}

func TestLoadSettings_EntryNotAMappingIsError(t *testing.T) {
	root := loadRoot(t, `
conjecture: 1.0.0
settings:
  x: 5
`)

	_, err := LoadSettings(root)
	if err == nil {
		t.Fatal("LoadSettings with a scalar setting entry = nil error, want one")
	}
}

func TestLoadSettings_MissingFloorIsError(t *testing.T) {
	root := loadRoot(t, `
conjecture: 1.0.0
settings:
  x:
    direction: higher
`)

	_, err := LoadSettings(root)
	if err == nil {
		t.Fatal("LoadSettings with a missing \"floor\" = nil error, want one")
	}
}

func TestLoadSettings_NonNumericFloorIsError(t *testing.T) {
	root := loadRoot(t, `
conjecture: 1.0.0
settings:
  x:
    floor: not-a-number
    direction: higher
`)

	_, err := LoadSettings(root)
	if err == nil {
		t.Fatal("LoadSettings with a non-numeric \"floor\" = nil error, want one")
	}
}

func TestLoadSettings_MissingDirectionIsError(t *testing.T) {
	root := loadRoot(t, `
conjecture: 1.0.0
settings:
  x:
    floor: 1
`)

	_, err := LoadSettings(root)
	if err == nil {
		t.Fatal("LoadSettings with a missing \"direction\" = nil error, want one")
	}
}

func TestLoadSettings_UnrecognizedDirectionIsError(t *testing.T) {
	root := loadRoot(t, `
conjecture: 1.0.0
settings:
  x:
    floor: 1
    direction: sideways
`)

	_, err := LoadSettings(root)
	if err == nil {
		t.Fatal("LoadSettings with an unrecognized \"direction\" = nil error, want one")
	}
}
