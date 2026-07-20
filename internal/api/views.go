package api

import (
	"context"
	"net/url"
)

type View struct {
	Name string `json:"name"`
}

type ViewsQuery struct {
	Brief bool
}

func (vq *ViewsQuery) encode(u *url.URL) string {
	q := u.Query()

	if vq.Brief {
		q.Set("brief", "true")
	}

	return q.Encode()
}

func (vq *ViewsQuery) GetViews(ctx context.Context, c *Client) ([]View, error) {
	u := c.netboxURL.JoinPath("views", "/")
	u.RawQuery = vq.encode(u)
	views, err := getMany[View](ctx, c, u.String())
	if err != nil {
		return nil, err
	}
	return views, nil
}
