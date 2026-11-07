// Package theme loads a data-only color file. It never executes a theme
// script. A live Omarchy watch is a later milestone.
package theme

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/richtext"
)

var hexColor = regexp.MustCompile(`^#[0-9a-fA-F]{6}$`)

// Tokens are optional hex colors. Zero values keep MyGo's window theme.
type Tokens struct {
	Accent  string `json:"accent,omitempty"`
	Surface string `json:"surface,omitempty"`
	Text    string `json:"text,omitempty"`
}

func Load(path string) (Tokens, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return Tokens{}, err
	}
	if strings.HasPrefix(strings.TrimSpace(string(b)), "#!") {
		return Tokens{}, errors.New("theme file is not data")
	}
	dec := json.NewDecoder(strings.NewReader(string(b)))
	dec.DisallowUnknownFields()
	var t Tokens
	if err := dec.Decode(&t); err != nil {
		return Tokens{}, err
	}
	for _, c := range []string{t.Accent, t.Surface, t.Text} {
		if c != "" && !hexColor.MatchString(c) {
			return Tokens{}, fmt.Errorf("color %q is not #rrggbb", c)
		}
	}
	return t, nil
}

func (t Tokens) Rich() *richtext.Theme {
	if t.Accent == "" && t.Text == "" {
		return nil
	}
	th := &richtext.Theme{}
	if t.Accent != "" {
		th.Link = ui.Hex(t.Accent)
	}
	if t.Text != "" {
		th.Text = ui.Hex(t.Text)
	}
	return th
}
