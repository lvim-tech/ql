package utils

import (
	"encoding/json"
	"fmt"
	"os/exec"
)

// WaylandActiveWindowGeometry returns the focused window's geometry as
// "x,y WxH" — the -g format grim, slurp and wf-recorder share. It speaks
// sway and Hyprland IPC; other wlroots compositors (mango, dwl, river)
// expose no focused-window query at all, so callers must be ready for an
// error and degrade — usually to fullscreen — rather than fail.
func WaylandActiveWindowGeometry() (string, error) {
	if CommandExists("swaymsg") {
		if out, err := exec.Command("swaymsg", "-t", "get_tree").Output(); err == nil {
			if geometry, ok := focusedSwayRect(out); ok {
				return geometry, nil
			}
		}
	}

	if CommandExists("hyprctl") {
		if out, err := exec.Command("hyprctl", "activewindow", "-j").Output(); err == nil {
			var window struct {
				At   [2]int `json:"at"`
				Size [2]int `json:"size"`
			}
			if err := json.Unmarshal(out, &window); err == nil && window.Size[0] > 0 {
				return fmt.Sprintf("%d,%d %dx%d", window.At[0], window.At[1], window.Size[0], window.Size[1]), nil
			}
		}
	}

	return "", fmt.Errorf("no compositor IPC for the active window")
}

// focusedSwayRect walks sway's layout tree for the focused node's rect.
func focusedSwayRect(tree []byte) (string, bool) {
	var node struct {
		Focused bool `json:"focused"`
		Rect    struct {
			X      int `json:"x"`
			Y      int `json:"y"`
			Width  int `json:"width"`
			Height int `json:"height"`
		} `json:"rect"`
		Nodes         []json.RawMessage `json:"nodes"`
		FloatingNodes []json.RawMessage `json:"floating_nodes"`
	}
	if json.Unmarshal(tree, &node) != nil {
		return "", false
	}
	if node.Focused {
		return fmt.Sprintf("%d,%d %dx%d", node.Rect.X, node.Rect.Y, node.Rect.Width, node.Rect.Height), true
	}
	for _, child := range append(node.Nodes, node.FloatingNodes...) {
		if geometry, ok := focusedSwayRect(child); ok {
			return geometry, true
		}
	}
	return "", false
}
