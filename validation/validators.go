package validation

import (
	"fmt"
	"regexp"
	"strings"
)

// StockValidator validates inventory levels
type StockValidator struct {
	minStock int
	maxStock int
}

// NewStockValidator creates a new stock validator
func NewStockValidator(min, max int) *StockValidator {
	if min < 0 {
		min = 0
	}
	if max <= 0 {
		max = 999999
	}
	return &StockValidator{minStock: min, maxStock: max}
}

// Validate checks if stock value is valid
func (sv *StockValidator) Validate(value int) error {
	if value < sv.minStock {
		return fmt.Errorf("stock cannot be negative, got %d", value)
	}
	if value > sv.maxStock {
		return fmt.Errorf("stock exceeds maximum of %d, got %d", sv.maxStock, value)
	}
	return nil
}

// ValidateAmount validates monetary amounts
func ValidateAmount(amount float64, minAmount, maxAmount float64) error {
	if amount < minAmount {
		return fmt.Errorf("amount must be at least %.2f, got %.2f", minAmount, amount)
	}
	if maxAmount > 0 && amount > maxAmount {
		return fmt.Errorf("amount must not exceed %.2f, got %.2f", maxAmount, amount)
	}
	return nil
}

// ValidateQuantity validates quantity values
func ValidateQuantity(quantity int, minQty, maxQty int) error {
	if quantity < minQty {
		return fmt.Errorf("quantity must be at least %d, got %d", minQty, quantity)
	}
	if maxQty > 0 && quantity > maxQty {
		return fmt.Errorf("quantity must not exceed %d, got %d", maxQty, quantity)
	}
	return nil
}

// ValidateEmail validates email format
func ValidateEmail(email string) error {
	if len(strings.TrimSpace(email)) == 0 {
		return fmt.Errorf("email cannot be empty")
	}

	// Simple email validation regex
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	if !emailRegex.MatchString(email) {
		return fmt.Errorf("invalid email format: %s", email)
	}
	return nil
}

// ValidatePhoneNumber validates phone number format
func ValidatePhoneNumber(phone string) error {
	phone = strings.TrimSpace(phone)
	if len(phone) == 0 {
		return fmt.Errorf("phone number cannot be empty")
	}

	// Remove common formatting characters
	cleanedPhone := regexp.MustCompile(`[\s\-\(\)\+]`).ReplaceAllString(phone, "")

	// Check if it's numeric and has valid length
	if !regexp.MustCompile(`^\d{10,15}$`).MatchString(cleanedPhone) {
		return fmt.Errorf("invalid phone number format: %s", phone)
	}
	return nil
}

// ValidateString validates string length and content
func ValidateString(value string, minLen, maxLen int, allowEmpty bool) error {
	if !allowEmpty && len(strings.TrimSpace(value)) == 0 {
		return fmt.Errorf("string cannot be empty")
	}

	if len(value) < minLen {
		return fmt.Errorf("string length must be at least %d characters, got %d", minLen, len(value))
	}

	if maxLen > 0 && len(value) > maxLen {
		return fmt.Errorf("string length must not exceed %d characters, got %d", maxLen, len(value))
	}

	return nil
}

// ValidateSeatRange validates ticket seat ranges
func ValidateSeatRange(seatStart, seatEnd int) error {
	if seatStart < 0 {
		return fmt.Errorf("seat start must be non-negative, got %d", seatStart)
	}

	if seatEnd <= 0 {
		return fmt.Errorf("seat end must be positive, got %d", seatEnd)
	}

	if seatStart >= seatEnd {
		return fmt.Errorf("seat start (%d) must be less than seat end (%d)", seatStart, seatEnd)
	}

	return nil
}

// ValidateTimeRange validates date/time ranges
func ValidateTimeRange(startTime, endTime int64) error {
	if startTime < 0 {
		return fmt.Errorf("start time cannot be negative")
	}

	if endTime <= startTime {
		return fmt.Errorf("end time must be after start time")
	}

	return nil
}

// ValidateIDFormat validates entity ID format
func ValidateIDFormat(id string) error {
	if len(strings.TrimSpace(id)) == 0 {
		return fmt.Errorf("ID cannot be empty")
	}

	if len(id) > 64 {
		return fmt.Errorf("ID too long: %d characters", len(id))
	}

	return nil
}

// ValidatePrice validates product prices
func ValidatePrice(price float64) error {
	if price <= 0 {
		return fmt.Errorf("price must be positive, got %.2f", price)
	}

	// Check for reasonable upper limit (e.g., 1 crore rupees max)
	const maxPrice = 10000000.0
	if price > maxPrice {
		return fmt.Errorf("price exceeds maximum of %.2f, got %.2f", maxPrice, price)
	}

	return nil
}

// NegativeStockPrevention checks if operations would result in negative stock
func NegativeStockPrevention(currentStock, requestedDeduction int) error {
	if currentStock < 0 {
		return fmt.Errorf("database stock is negative: %d", currentStock)
	}

	resultingStock := currentStock - requestedDeduction
	if resultingStock < 0 {
		return fmt.Errorf("operation would result in negative stock: %d - %d = %d", currentStock, requestedDeduction, resultingStock)
	}

	return nil
}
