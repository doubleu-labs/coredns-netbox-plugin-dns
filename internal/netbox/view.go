package netbox

import (
	"slices"

	internal "github.com/doubleu-labs/coredns-netbox-plugin-dns/internal/view"
)

type view struct {
	Name string `json:"name"`
}

func GetViews(c *Client, v *internal.View) (
	found []string,
	unknown []string,
	err error,
) {
	u := c.NetboxURL.JoinPath("views", "/")
	q := u.Query()
	q.Set("brief", "true")
	for _, n := range v.Include {
		q.Add("name", n)
	}
	for _, n := range v.Exclude {
		q.Add("name", n)
	}
	u.RawQuery = q.Encode()
	avs, err := getMany[view](c, u.String())
	if err != nil {
		return nil, nil, err
	}
	for _, av := range avs {
		if slices.Contains(v.Include, av.Name) {
			found = append(found, av.Name)
		} else {
			unknown = append(unknown, av.Name)
		}
		if slices.Contains(v.Exclude, av.Name) {
			found = append(found, av.Name)
		} else {
			unknown = append(unknown, av.Name)
		}
	}
	return
}
