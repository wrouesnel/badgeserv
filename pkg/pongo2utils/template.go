package pongo2utils

import (
	"github.com/flosch/pongo2/v6"
	"github.com/pkg/errors"
	"github.com/samber/lo"
)

type Template struct {
	*pongo2.Template
}

// Decode implements the kong.MapperValue on-top of pongo2 Templates.
func (t *Template) UnmarshalText(text []byte) error {
	loadedTemplate, err := pongo2.FromBytes(text)
	if loadedTemplate == nil {
		t.Template = lo.Must(pongo2.FromBytes([]byte("")))
	} else {
		t.Template = loadedTemplate
	}
	if err != nil {
		return errors.Wrap(err, "UnmarshalText")
	}

	return nil
}
