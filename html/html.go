package html

import (
	h "github.com/chasefleming/elem-go" //nolint
	a "github.com/chasefleming/elem-go/attrs"
)

var (
	dateFormat     string = "Monday 02. January 2006"
	dateTimeFormat string = "Monday 02. January 2006 15:04"
)

func Base(props a.Props, children ...h.Node) *h.Element {
	content := h.Html(a.Props{
		a.Lang: "en",
	},
		h.Head(nil,
			h.Meta(a.Props{
				a.Charset: "utf-8",
			}),
			h.Meta(a.Props{
				a.Name:    "viewport",
				a.Content: "initial-scale=1,maximum-scale=1,user-scalable=no",
			}),
			h.Title(nil, h.Text("kradalby.no")),
			// h.Link(a.Props{
			// 	a.Rel:  "stylesheet",
			// 	a.Href: "static/tailwind.css",
			// }),
		),
		h.Body(props,
			children...,
		),
	)

	return content
}

func Page() *h.Element {
	return Base(nil,
		h.Div(
			a.Props{
				a.Class: "",
			},
			h.Nav(
				a.Props{},
				h.A(
					a.Props{
						a.Href: "/",
					},
					h.Span(
						a.Props{
							a.Class: "",
						},
						// h.Img(
						// 	a.Props{
						// 		a.Class: "h-12 md:h-16 mr-4",
						// 		a.Src:   "./static/location.svg",
						// 	}),
						h.H1(
							a.Props{
								a.Class: "",
							}, h.Text("kradalby.no")),
					),
				)),
			h.Main(
				nil,
				h.Text("Hellow"),
			),
			h.Footer(
				a.Props{
					a.Class: "",
				},
			),
		),
	)
}
