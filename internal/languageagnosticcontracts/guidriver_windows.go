package languageagnosticcontracts

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

// NewNativeDriver returns a [Driver] backed by Windows' own real UI
// Automation, observing the first top-level window belonging to the
// named process (e.g. "notepad.exe" - matched the same way
// [AutomationElement.ProcessIdProperty] already requires: by the
// target's own running process, not a window title, which a caller
// may not know in advance or which may change).
func NewNativeDriver(processName string) Driver {
	return &WindowsDriver{ProcessName: processName}
}

// WindowsDriver observes a specific running Windows application's
// accessibility tree by shelling out to PowerShell's own
// System.Windows.Automation assembly (see guidriver.go's own doc
// comment for why: the same managed client over the real native
// IUIAutomation COM server, reached as an external tool rather than
// through hand-rolled COM vtable calls).
type WindowsDriver struct {
	// ProcessName is the target process's own executable name (e.g.
	// "notepad.exe").
	ProcessName string
	// run is a test hook - a real caller leaves it nil, so Snapshot
	// calls [runPowerShellSnapshot] (the real "powershell.exe"
	// invocation); a test substitutes a canned response instead of a
	// real process and a real desktop session.
	run func(ctx context.Context, script string) ([]byte, error)
}

// Snapshot implements [Driver] for a real Windows target.
func (d *WindowsDriver) Snapshot(ctx context.Context) (GUIElement, error) {
	run := d.run
	if run == nil {
		run = runPowerShellSnapshot
	}
	out, err := run(ctx, windowsSnapshotScript(d.ProcessName))
	if err != nil {
		return GUIElement{}, fmt.Errorf("languageagnosticcontracts: %w", err)
	}
	element, err := parseWindowsSnapshot(out)
	if err != nil {
		return GUIElement{}, fmt.Errorf("languageagnosticcontracts: %w", err)
	}
	return element, nil
}

// runPowerShellSnapshot runs script as a real "powershell.exe"
// invocation and returns its stdout.
func runPowerShellSnapshot(ctx context.Context, script string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, "powershell.exe", "-NoProfile", "-NonInteractive", "-Command", script)
	out, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("powershell.exe failed: %w (stderr: %s)", err, string(exitErr.Stderr))
		}
		return nil, fmt.Errorf("running powershell.exe: %w", err)
	}
	return out, nil
}

// windowsSnapshotScript renders the PowerShell script that walks the
// named process's own top-level window through System.Windows.
// Automation's TreeWalker (ControlViewWalker - the same "coarse
// grain" view [Role] itself already commits to, skipping raw-view
// noise no native accessibility API's own coarser callers see
// either), and prints the whole tree as JSON. processName is shell-
// quoted as a single-quoted PowerShell string literal - the only
// escaping a literal process name (no apostrophes in a real .exe
// filename) needs.
func windowsSnapshotScript(processName string) string {
	quoted := "'" + strings.ReplaceAll(processName, "'", "''") + "'"
	return `
Add-Type -AssemblyName UIAutomationClient
Add-Type -AssemblyName UIAutomationTypes

$proc = Get-Process -Name ([System.IO.Path]::GetFileNameWithoutExtension(` + quoted + `)) -ErrorAction Stop | Select-Object -First 1
$root = [System.Windows.Automation.AutomationElement]::RootElement
$cond = New-Object System.Windows.Automation.PropertyCondition([System.Windows.Automation.AutomationElement]::ProcessIdProperty, $proc.Id)
$window = $root.FindFirst([System.Windows.Automation.TreeScope]::Children, $cond)
if ($null -eq $window) {
    throw "no top-level window found for process $(` + quoted + `)"
}

function Convert-Element($el) {
    $walker = [System.Windows.Automation.TreeWalker]::ControlViewWalker
    $children = New-Object System.Collections.ArrayList
    $child = $walker.GetFirstChild($el)
    while ($null -ne $child) {
        [void]$children.Add((Convert-Element $child))
        $child = $walker.GetNextSibling($child)
    }
    $value = ""
    try {
        $vp = $el.GetCurrentPattern([System.Windows.Automation.ValuePattern]::Pattern)
        $value = $vp.Current.Value
    } catch {}
    [PSCustomObject]@{
        Role     = $el.Current.ControlType.ProgrammaticName
        Name     = $el.Current.Name
        Value    = $value
        Children = @($children.ToArray())
    }
}

Convert-Element $window | ConvertTo-Json -Depth 100 -Compress
`
}

// windowsElement mirrors windowsSnapshotScript's own JSON shape.
// Children is decoded as [json.RawMessage] rather than
// []windowsElement directly: PowerShell's ConvertTo-Json renders a
// single-element array as a bare JSON object rather than a one-
// element array in some versions (an observed, undocumented quirk,
// not this package's own choice), so [windowsElement.children] must
// accept either shape.
type windowsElement struct {
	Role     string          `json:"Role"`
	Name     string          `json:"Name"`
	Value    string          `json:"Value"`
	Children json.RawMessage `json:"Children"`
}

// children decodes [windowsElement.Children] as either a JSON array
// or (PowerShell's own single-element quirk) a bare JSON object - see
// [windowsElement]'s own doc comment.
func (e windowsElement) children() ([]windowsElement, error) {
	if len(e.Children) == 0 || string(e.Children) == "null" {
		return nil, nil
	}
	var asSlice []windowsElement
	if err := json.Unmarshal(e.Children, &asSlice); err == nil {
		return asSlice, nil
	}
	var asSingle windowsElement
	if err := json.Unmarshal(e.Children, &asSingle); err != nil {
		return nil, fmt.Errorf("decoding Children: %w", err)
	}
	return []windowsElement{asSingle}, nil
}

// windowsControlTypeRole maps UI Automation's own ControlType.
// ProgrammaticName values (e.g. "ControlType.Button") to [Role] - the
// closed set [Role] itself already commits to; every ProgrammaticName
// this does not recognize maps to [UnknownRole], the same "not every
// role any one native API enumerates" posture [Role]'s own doc
// comment already takes.
var windowsControlTypeRole = map[string]Role{
	"ControlType.Window":   Window,
	"ControlType.Button":   Button,
	"ControlType.Edit":     TextField,
	"ControlType.Text":     Label,
	"ControlType.CheckBox": CheckBox,
	"ControlType.MenuItem": MenuItem,
}

// parseWindowsSnapshot decodes windowsSnapshotScript's own JSON
// output into a [GUIElement] tree - pure, and so unit-testable against
// canned JSON with no real PowerShell process or desktop session
// involved.
func parseWindowsSnapshot(data []byte) (GUIElement, error) {
	var root windowsElement
	if err := json.Unmarshal(data, &root); err != nil {
		return GUIElement{}, fmt.Errorf("decoding PowerShell output: %w", err)
	}
	return convertWindowsElement(root)
}

func convertWindowsElement(e windowsElement) (GUIElement, error) {
	children, err := e.children()
	if err != nil {
		return GUIElement{}, err
	}
	role, ok := windowsControlTypeRole[e.Role]
	if !ok {
		role = UnknownRole
	}
	element := GUIElement{Role: role, Name: e.Name, Value: e.Value}
	for _, child := range children {
		converted, err := convertWindowsElement(child)
		if err != nil {
			return GUIElement{}, err
		}
		element.Children = append(element.Children, converted)
	}
	return element, nil
}
