// guidriver.go defines the "Generic accessibility driver for GUI
// targets" contract: "For a GUI target, the system SHALL use a
// generic accessibility-based driver to observe behavior, rather
// than a framework-specific UI automation integration... WHEN the
// target under test is a GUI application THEN the system observes
// its behavior through the generic accessibility driver."
//
// [Driver] itself is this contract, OS-agnostic by construction - a
// caller always sees the same interface regardless of platform.
// [NewNativeDriver] resolves it to one real, OS-specific backend per
// build (guidriver_windows.go for Windows' own UI Automation,
// guidriver_linux.go for Linux's own AT-SPI, guidriver_unsupported.go
// everywhere else, selected automatically by Go's own per-file GOOS
// filename convention - never by this file, which has no OS-specific
// code of its own). Neither backend hand-rolls its native API's own
// COM/D-Bus wire protocol beyond what each already demands: the
// Windows backend shells out to PowerShell's own managed UI Automation
// client (the same real native IUIAutomation COM server underneath,
// reached the same way this project already reaches Docker and z3 -
// a real external implementation of a real standard, not a
// reimplementation of one); the Linux backend speaks AT-SPI's own
// D-Bus protocol directly via a well-maintained D-Bus client library,
// since D-Bus method calls carry none of raw COM's vtable/ABI risk.
//
// Honesty note: only the Windows and Linux backends are verified for
// real (this project has a real Windows machine and a real WSL Linux
// environment to test against, this session) - macOS's own
// Accessibility API has no backend yet (guidriver_unsupported.go
// covers it with a clear runtime error, not a guess at its API).
// Neither backend's own live integration test runs in this project's
// CI (ubuntu-latest has no desktop session, and compiles only the
// Linux-tagged file in the first place) - each backend's own pure
// tree-building/role-mapping logic is unit-tested against a fake
// instead, and its real end-to-end behavior is verified manually
// against a real running GUI target.
//
// Nothing in this codebase calls [Driver] yet: the cjtr-language
// grammar this package parses has no way to declare a rule's target
// as a GUI application, distinct from a CLI or HTTP one, in the first
// place - that is its own separate, ungrounded gap, not something
// this file invents a grammar extension for.
package languageagnosticcontracts

import "context"

// Role is one GUI element's accessibility role - the one property
// shared, under slightly different names, by every major native
// accessibility API this driver could eventually be backed by: UI
// Automation's own ControlType, AT-SPI's own Role, and macOS
// Accessibility's own AXRole. This is the closed set every one of
// those three already agrees on at the coarse grain "observing
// behavior" needs, not every role any one of them itself enumerates
// (each has dozens more, many specific to one platform's own widget
// set).
type Role int

const (
	UnknownRole Role = iota
	Window
	Button
	TextField
	Label
	CheckBox
	MenuItem
)

func (r Role) String() string {
	switch r {
	case Window:
		return "window"
	case Button:
		return "button"
	case TextField:
		return "text field"
	case Label:
		return "label"
	case CheckBox:
		return "check box"
	case MenuItem:
		return "menu item"
	default:
		return "unknown"
	}
}

// GUIElement is one accessible element's own observed state - Role,
// Name (its own accessible name/label - UI Automation's Name
// property, AT-SPI's Name property, AX's AXTitle/AXDescription), and
// Value (its own current value, when it has one - a text field's
// text, say) - the three properties every one of the three native
// APIs already exposes for any element, regardless of which specific
// widget toolkit built it. Children is every element nested under
// this one in the accessibility tree.
//
// Read-only by construction, the same posture this capability's own
// "Observation only through declared contracts" already takes: this
// type only ever holds what a driver read from an already-running
// target's accessibility tree, never an action to perform on it
// (click, type, …) - [Driver] itself has no method for one either.
type GUIElement struct {
	Role     Role
	Name     string
	Value    string
	Children []GUIElement
}

// Driver observes a GUI target's current accessibility tree - the
// "generic accessibility-based driver" the specification names, as
// opposed to a framework-specific UI automation integration (a
// WinAppDriver/Appium-style per-framework tool). A real
// implementation backs this with one OS's own native accessibility
// API; this package does not provide one yet (see this file's own
// doc comment for why).
type Driver interface {
	// Snapshot returns the target application's own current
	// accessibility tree, rooted at its top-level window.
	Snapshot(ctx context.Context) (GUIElement, error)
}

// FindByName searches root's own accessibility tree (itself and
// every descendant), depth-first, for the first element whose Name
// matches name exactly - the basic lookup "observe behavior through
// the accessibility driver" needs before anything else: finding the
// one element a check is actually about, out of a whole window's own
// tree, found reports whether one was found at all.
func FindByName(root GUIElement, name string) (element GUIElement, found bool) {
	if root.Name == name {
		return root, true
	}
	for _, child := range root.Children {
		if found, ok := FindByName(child, name); ok {
			return found, true
		}
	}
	return GUIElement{}, false
}
