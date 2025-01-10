package whatsapp

import (
	"sort"
	"strings"

	"github.com/nyaruka/courier"
	"github.com/nyaruka/courier/handlers"
	"golang.org/x/exp/maps"
)

func GetTemplatePayload(templating *courier.Templating) *Template {
	template := &Template{
		Name:       templating.Template.Name,
		Language:   &Language{Policy: "deterministic", Code: templating.Language},
		Components: []*Component{},
	}

	for _, comp := range templating.Components {
		// get the variables used by this component in order of their names 1, 2 etc
		compParams := make([]courier.TemplatingVariable, 0, len(comp.Variables))
		varNames := maps.Keys(comp.Variables)
		sort.Strings(varNames)
		for _, varName := range varNames {
			compParams = append(compParams, templating.Variables[comp.Variables[varName]])
		}

		var component *Component

		if comp.Type == "header" || strings.HasPrefix(comp.Type, "header/") {
			component = &Component{Type: "header"}

			for _, p := range compParams {
				if p.Type != "text" {
					attType, attURL := handlers.SplitAttachment(p.Value)
					attType = strings.Split(attType, "/")[0]

					if attType == "image" {
						component.Params = append(component.Params, &Param{Type: "image", Image: &Media{Link: attURL}})
					} else if attType == "video" {
						component.Params = append(component.Params, &Param{Type: "video", Video: &Media{Link: attURL}})
					} else if attType == "application" {
						component.Params = append(component.Params, &Param{Type: "document", Document: &Media{Link: attURL}})
					}
				} else {
					component.Params = append(component.Params, &Param{Type: p.Type, Text: p.Value})
				}
			}
		} else if comp.Type == "body" || strings.HasPrefix(comp.Type, "body/") {
			component = &Component{Type: "body"}

			for _, p := range compParams {
				component.Params = append(component.Params, &Param{Type: p.Type, Text: p.Value})
			}
		} else if strings.HasPrefix(comp.Type, "button/") {
			component = &Component{Type: "button", Index: strings.TrimPrefix(comp.Name, "button."), SubType: strings.TrimPrefix(comp.Type, "button/"), Params: []*Param{}}

			for _, p := range compParams {
				if comp.Type == "button/url" {
					component.Params = append(component.Params, &Param{Type: "text", Text: p.Value})
				} else {
					component.Params = append(component.Params, &Param{Type: "payload", Payload: p.Value})
				}
			}
		}

		if component != nil {
			template.Components = append(template.Components, component)
		}
	}

	return template
}
