package pongorenderer

import (
	"io"

	"github.com/flosch/pongo2/v6"
	"github.com/labstack/echo/v4"
	"github.com/pkg/errors"
)

type Renderer struct {
	templateSet *pongo2.TemplateSet
}

func NewRenderer(templateSet *pongo2.TemplateSet) Renderer {
	return Renderer{
		templateSet: templateSet,
	}
}

// Render impements echo.Renderer. Pongo2 context data is placed under the prefix "t"
// for access within templates.
func (r Renderer) Render(writer io.Writer, templateName string, templateData interface{}, context echo.Context) error {
	template, err := r.templateSet.FromCache(templateName)
	if err != nil {
		return errors.Wrapf(err, "Render: loading template failed %s", templateName)
	}

	templateContext := map[string]interface{}{}
	templateContext["t"] = templateData

	return errors.Wrapf(template.ExecuteWriter(templateContext, context.Response().Writer), "Render: template execution failed %s", templateName)
}
