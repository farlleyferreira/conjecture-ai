package languageagnosticcontracts

import (
	"context"
	"fmt"
	"strings"

	"github.com/godbus/dbus/v5"
)

// NewNativeDriver returns a [Driver] backed by Linux's own real
// AT-SPI accessibility bus, observing the first registered
// application whose own accessible Name contains appName (case-
// insensitive; "" matches whichever single application is
// registered, and is an error if more than one is).
func NewNativeDriver(appName string) Driver {
	return &LinuxDriver{AppName: appName}
}

// atspiAccessibleIface and atspiTextIface name the two AT-SPI D-Bus
// interfaces this driver calls - "org.a11y.atspi.Accessible" (every
// accessible object's own tree/identity methods) and
// "org.a11y.atspi.Text" (present only on an object that actually
// holds text, e.g. a label or a text entry - see [dbusAccessible.
// Text]'s own doc comment for why a missing Text interface is not
// treated as an error).
const (
	atspiAccessibleIface = "org.a11y.atspi.Accessible"
	atspiTextIface       = "org.a11y.atspi.Text"
	// atspiRegistryDest is the at-spi2-core registry daemon's own
	// well-known bus name on the real accessibility bus - the one
	// fixed address every registered application's own root
	// accessible is reachable from.
	atspiRegistryDest = "org.a11y.atspi.Registry"
	atspiRootPath     = dbus.ObjectPath("/org/a11y/atspi/accessible/root")
)

// atspiAccessible is the minimal real AT-SPI surface this driver
// needs from one accessible object, abstracted behind an interface so
// [atspiSnapshot] - the actual tree-walking/role-mapping logic - is
// unit-testable against a fake instead of a real D-Bus connection and
// a real desktop session.
type atspiAccessible interface {
	Name() (string, error)
	RoleName() (string, error)
	// Text returns this accessible's own text content ("" if it has
	// none, or does not support AT-SPI's own Text interface at all -
	// see [dbusAccessible.Text]).
	Text() (string, error)
	Children() ([]atspiAccessible, error)
}

// atspiRoleRole maps AT-SPI's own GetRoleName() strings to [Role] -
// verified against this package's own real test run against a real
// GTK3 application under a real AT-SPI bus (this WSL environment's
// own at-spi2-core 2.60.4), not guessed from documentation alone: a
// real GtkButton reported role "button" and a real GtkEntry reported
// role "text", for instance - not the "push button"/"entry" ATK
// documentation examples might suggest, which this mapping also keeps
// alongside the observed real values, since another toolkit or
// at-spi2-core version may still use them. This is the same closed-
// set, "not every role any native API enumerates" posture [Role]'s
// own doc comment already takes for every backend.
var atspiRoleRole = map[string]Role{
	"frame":       Window,
	"window":      Window,
	"button":      Button,
	"push button": Button,
	"label":       Label,
	"entry":       TextField,
	"text":        TextField,
	"check box":   CheckBox,
	"menu item":   MenuItem,
}

// atspiSnapshot converts root's own real AT-SPI tree into a
// [GUIElement] tree - pure, given anything implementing
// [atspiAccessible], so it is unit-testable with no real D-Bus
// connection involved.
func atspiSnapshot(root atspiAccessible) (GUIElement, error) {
	name, err := root.Name()
	if err != nil {
		return GUIElement{}, fmt.Errorf("getting name: %w", err)
	}
	roleName, err := root.RoleName()
	if err != nil {
		return GUIElement{}, fmt.Errorf("getting role: %w", err)
	}
	value, err := root.Text()
	if err != nil {
		return GUIElement{}, fmt.Errorf("getting text: %w", err)
	}
	children, err := root.Children()
	if err != nil {
		return GUIElement{}, fmt.Errorf("getting children: %w", err)
	}

	role, ok := atspiRoleRole[roleName]
	if !ok {
		role = UnknownRole
	}
	element := GUIElement{Role: role, Name: name, Value: value}
	for _, child := range children {
		converted, err := atspiSnapshot(child)
		if err != nil {
			return GUIElement{}, err
		}
		element.Children = append(element.Children, converted)
	}
	return element, nil
}

// LinuxDriver observes a specific registered application's real
// accessibility tree over AT-SPI's own D-Bus protocol (see
// guidriver.go's own doc comment for why this talks D-Bus directly
// rather than through a native COM-style binding: D-Bus method calls
// carry none of raw COM's vtable/ABI risk).
type LinuxDriver struct {
	// AppName matches a registered application's own accessible Name
	// (case-insensitive substring) - see [NewNativeDriver]'s own doc
	// comment.
	AppName string
}

// atspiRef mirrors AT-SPI's own "(so)" struct wire type: a reference
// to one accessible object as the (bus name, object path) pair that
// actually addresses it - necessary because, unlike a single COM or
// AT-SPI server, each registered *application* on the accessibility
// bus is its own separate D-Bus service name, not merely a different
// object path under one shared name.
type atspiRef struct {
	Name string
	Path dbus.ObjectPath
}

// dbusAccessible is [atspiAccessible]'s own real implementation, one
// D-Bus call per method - the real accessibility-bus wiring
// [atspiSnapshot] itself never calls directly.
type dbusAccessible struct {
	conn *dbus.Conn
	dest string
	path dbus.ObjectPath
}

func (a dbusAccessible) object() dbus.BusObject {
	return a.conn.Object(a.dest, a.path)
}

func (a dbusAccessible) Name() (string, error) {
	var v dbus.Variant
	if err := a.object().Call("org.freedesktop.DBus.Properties.Get", 0, atspiAccessibleIface, "Name").Store(&v); err != nil {
		return "", err
	}
	name, _ := v.Value().(string)
	return name, nil
}

func (a dbusAccessible) RoleName() (string, error) {
	var role string
	if err := a.object().Call(atspiAccessibleIface+".GetRoleName", 0).Store(&role); err != nil {
		return "", err
	}
	return role, nil
}

// Text returns GetText(0, -1) - AT-SPI's own "the whole text" call -
// from the real org.a11y.atspi.Text interface. A D-Bus error here
// (UnknownInterface/UnknownMethod, the normal outcome for an
// accessible that simply has no text, e.g. a plain button) is not
// reported as this method's own error: [GUIElement.Value] is already
// "" for an element with none, the same "Value, when it has one" the
// type's own doc comment already allows for.
func (a dbusAccessible) Text() (string, error) {
	var text string
	if err := a.object().Call(atspiTextIface+".GetText", 0, int32(0), int32(-1)).Store(&text); err != nil {
		return "", nil
	}
	return text, nil
}

func (a dbusAccessible) Children() ([]atspiAccessible, error) {
	var refs []atspiRef
	if err := a.object().Call(atspiAccessibleIface+".GetChildren", 0).Store(&refs); err != nil {
		return nil, err
	}
	children := make([]atspiAccessible, len(refs))
	for i, ref := range refs {
		children[i] = dbusAccessible{conn: a.conn, dest: ref.Name, path: ref.Path}
	}
	return children, nil
}

// Snapshot implements [Driver] for a real Linux target.
func (d *LinuxDriver) Snapshot(ctx context.Context) (GUIElement, error) {
	conn, err := connectA11yBus(ctx)
	if err != nil {
		return GUIElement{}, fmt.Errorf("languageagnosticcontracts: %w", err)
	}
	defer func() { _ = conn.Close() }()

	registryRoot := dbusAccessible{conn: conn, dest: atspiRegistryDest, path: atspiRootPath}
	apps, err := registryRoot.Children()
	if err != nil {
		return GUIElement{}, fmt.Errorf("languageagnosticcontracts: listing registered applications: %w", err)
	}

	app, err := selectApp(apps, d.AppName)
	if err != nil {
		return GUIElement{}, fmt.Errorf("languageagnosticcontracts: %w", err)
	}

	root, err := topLevelWindow(app)
	if err != nil {
		return GUIElement{}, fmt.Errorf("languageagnosticcontracts: %w", err)
	}

	element, err := atspiSnapshot(root)
	if err != nil {
		return GUIElement{}, fmt.Errorf("languageagnosticcontracts: %w", err)
	}
	return element, nil
}

// connectA11yBus connects to the real per-session accessibility bus -
// a separate bus from the ordinary session bus, whose own address is
// only discoverable by first asking the session bus for it (AT-SPI's
// own real, two-step bootstrap: "org.a11y.Bus"/"GetAddress" on the
// session bus, then connect to whatever address it returns).
func connectA11yBus(ctx context.Context) (*dbus.Conn, error) {
	sessionConn, err := dbus.ConnectSessionBus(dbus.WithContext(ctx))
	if err != nil {
		return nil, fmt.Errorf("connecting to the session bus: %w", err)
	}
	defer func() { _ = sessionConn.Close() }()

	var address string
	if err := sessionConn.Object("org.a11y.Bus", dbus.ObjectPath("/org/a11y/bus")).
		Call("org.a11y.Bus.GetAddress", 0).Store(&address); err != nil {
		return nil, fmt.Errorf("getting the accessibility bus address: %w", err)
	}

	a11yConn, err := dbus.Connect(address, dbus.WithContext(ctx))
	if err != nil {
		return nil, fmt.Errorf("connecting to the accessibility bus: %w", err)
	}
	return a11yConn, nil
}

// selectApp picks the one registered application [LinuxDriver.
// Snapshot] should observe - see [LinuxDriver.AppName]'s own doc
// comment.
func selectApp(apps []atspiAccessible, appName string) (atspiAccessible, error) {
	if appName == "" {
		if len(apps) == 1 {
			return apps[0], nil
		}
		return nil, fmt.Errorf("%d applications are registered and AppName is empty - name one", len(apps))
	}
	for _, app := range apps {
		name, err := app.Name()
		if err != nil {
			return nil, err
		}
		if strings.Contains(strings.ToLower(name), strings.ToLower(appName)) {
			return app, nil
		}
	}
	return nil, fmt.Errorf("no registered application matches %q", appName)
}

// topLevelWindow finds app's own first "frame"/"window"-roled child -
// the real top-level window [Driver.Snapshot]'s own contract wants
// rooted at, rather than the application object itself (which AT-SPI
// also exposes, one level above any actual window). Falls back to app
// itself if none is found, rather than failing: some minimal toolkits
// expose no separate frame object.
func topLevelWindow(app atspiAccessible) (atspiAccessible, error) {
	children, err := app.Children()
	if err != nil {
		return nil, err
	}
	for _, child := range children {
		roleName, err := child.RoleName()
		if err != nil {
			return nil, err
		}
		if roleName == "frame" || roleName == "window" {
			return child, nil
		}
	}
	return app, nil
}
