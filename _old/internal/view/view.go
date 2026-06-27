package view

import (
	"slices"
)

type View struct {
	Include []string
	Exclude []string
}

func New(i, e []string) *View {
	i = slices.DeleteFunc(
		i,
		func(s string) bool {
			return slices.Contains(e, s)
		},
	)
	return &View{
		Include: i,
		Exclude: e,
	}
}
