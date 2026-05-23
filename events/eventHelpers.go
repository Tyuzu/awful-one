package events

import (
	"naevis/models"
)

// toSafeEvent ensures no nil slices or zero-values, computes Prices & Currency.
func toSafeEvent(e models.Event) models.Event {
	// default empty slices
	if e.Tickets == nil {
		e.Tickets = []models.Ticket{}
	}
	if e.Merch == nil {
		e.Merch = []models.Merch{}
	}
	if e.FAQs == nil {
		e.FAQs = []models.FAQ{}
	}
	if e.Artists == nil {
		e.Artists = []string{}
	}
	if e.Tags == nil {
		e.Tags = []string{}
	}
	if e.HiredVendors == nil {
		e.HiredVendors = []models.VendorHiring{}
	}

	// sanitize zero dates
	if !e.Date.IsZero() {
		e.Date = e.Date.UTC()
	}
	if !e.CreatedAt.IsZero() {
		e.CreatedAt = e.CreatedAt.UTC()
	}
	if !e.UpdatedAt.IsZero() {
		e.UpdatedAt = e.UpdatedAt.UTC()
	}

	// compute prices & currency
	var prices []float64
	var currency string
	if len(e.Tickets) > 0 {
		for _, t := range e.Tickets {
			prices = append(prices, float64(t.Price)/100) // Convert paise to rupees (float64)
			if currency == "" && t.Currency != "" {
				currency = t.Currency
			}
		}
	} else {
		prices = []float64{0}
	}
	e.Prices = prices
	e.Currency = currency

	return e
}
