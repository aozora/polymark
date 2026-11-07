package model

import "github.com/egoist/mygo/ui/richtext"

// Aliases keep tests from importing the editor types only to pass zero
// attributes. Ordinary edits go through Document.Edit, never SetText.
type richtextEdit = richtext.Edit

func attr() richtext.Attributes { return richtext.Attributes{} }
