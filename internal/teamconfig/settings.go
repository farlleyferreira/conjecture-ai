package teamconfig

import (
	"fmt"
	"strconv"

	"github.com/farlleyferreira/conjecture-ai/internal/yamlstrict"
)

// Setting is one entry of a settings.cjtr file's own "settings"
// mapping: the team's floor for that setting, and which direction
// counts as stricter (see [Direction]).
type Setting struct {
	Floor     float64
	Direction Direction
}

// LoadSettings reads a strictly-loaded settings.cjtr document's
// top-level "settings" mapping - team-config's own "versioned,
// git-tracked team settings" requirement. A file with no "settings"
// key returns (nil, nil): not every settings.cjtr file is required to
// govern every setting this project knows about.
//
// Each entry names a floor (the value a setting's own current value
// is compared against, read as a decimal - exact enough for every
// setting this project has today: case counts, a 0-1 aging rate, a
// duration in seconds, enumeration caps) and a direction ("higher" or
// "lower", [Direction]'s own two values) - the same two facts
// [CheckFloor] itself already takes as explicit parameters, now read
// from a real file instead of a caller's own flags.
func LoadSettings(root *yamlstrict.Node) (map[string]Setting, error) {
	node, ok := root.Get("settings")
	if !ok {
		return nil, nil
	}
	if node.Kind != yamlstrict.MappingKind {
		return nil, fmt.Errorf("teamconfig: \"settings\" must be a mapping of name to {floor, direction}, got kind %v", node.Kind)
	}

	settings := make(map[string]Setting, len(node.Pairs))
	for _, p := range node.Pairs {
		if p.Value.Kind != yamlstrict.MappingKind {
			return nil, fmt.Errorf("teamconfig: setting %q must be a mapping with \"floor\" and \"direction\"", p.Key)
		}

		floorNode, ok := p.Value.Get("floor")
		if !ok || floorNode.Kind != yamlstrict.ScalarKind {
			return nil, fmt.Errorf("teamconfig: setting %q is missing a scalar \"floor\"", p.Key)
		}
		floor, err := strconv.ParseFloat(floorNode.Value, 64)
		if err != nil {
			return nil, fmt.Errorf("teamconfig: setting %q: \"floor\" %q is not a number: %w", p.Key, floorNode.Value, err)
		}

		directionNode, ok := p.Value.Get("direction")
		if !ok || directionNode.Kind != yamlstrict.ScalarKind {
			return nil, fmt.Errorf("teamconfig: setting %q is missing a scalar \"direction\"", p.Key)
		}
		var direction Direction
		switch directionNode.Value {
		case "higher":
			direction = Higher
		case "lower":
			direction = Lower
		default:
			return nil, fmt.Errorf("teamconfig: setting %q: \"direction\" must be \"higher\" or \"lower\", got %q", p.Key, directionNode.Value)
		}

		settings[p.Key] = Setting{Floor: floor, Direction: direction}
	}
	return settings, nil
}
