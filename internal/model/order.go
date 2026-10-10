package model

import "slices"

// Reordered moves an event onto an anchor's position in the visible order.
// IDs, not UI row indices, identify both events. Moving down inserts after
// the anchor; moving up inserts before it.
func (c Config) Reordered(id, anchorID string) (Config, bool) {
	items := c.Ordered()
	from := slices.IndexFunc(items, func(item Countdown) bool { return item.ID == id })
	to := slices.IndexFunc(items, func(item Countdown) bool { return item.ID == anchorID })
	if from < 0 || to < 0 || from == to {
		return c, false
	}
	item := items[from]
	items = slices.Delete(items, from, from+1)
	items = slices.Insert(items, to, item)
	next := c
	next.Sort, next.Countdowns = "manual", items
	return next, true
}
