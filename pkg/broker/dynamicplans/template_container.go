package dynamicplans

import (
	"encoding/json"
	"text/template"
)

// Helper type to get a better template representation in JSON
type TemplateContainer struct {
	*template.Template
}

func (t TemplateContainer) String() string {
	return t.Template.Root.String()
}

func (t TemplateContainer) MarshalJSON() ([]byte, error) {
	return json.Marshal(t.String())
}
