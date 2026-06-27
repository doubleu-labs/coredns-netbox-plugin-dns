package catalog

type Catalog struct {
	Poller *Poller
}

func New(
	pc *PollerConfig,
) *Catalog {
	c := &Catalog{}
	c.Poller = &Poller{
		PollerConfig: *pc,
		catalog:      c,
	}
	return c
}
