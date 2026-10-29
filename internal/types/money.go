package types

import (
	"database/sql/driver"
	"strings"

	"github.com/shopspring/decimal"
)

// Money handling mirrors the legacy use of java.math.BigDecimal. shopspring/decimal
// provides arbitrary-precision decimal arithmetic so discount and price math stays
// exact.
//
// shopspring v1.6 trims trailing zeros in String()/MarshalJSON (100.00 -> "100"),
// whereas the legacy Jackson+BigDecimal serialisation preserved the scale and
// emitted 100.00. The Money type below wraps decimal and always serialises with
// exactly two decimal places so monetary REST/RPC payloads match the old system.
func init() {
	decimal.MarshalJSONWithoutQuotes = true
}

// Money is a decimal amount that always serialises with two decimal places
// (e.g. 100.00), matching the legacy BigDecimal output. It embeds decimal.Decimal
// so all arithmetic methods are promoted; arithmetic results are plain decimals
// that callers wrap back into Money at the serialisation boundary.
type Money struct {
	decimal.Decimal
}

// NewMoney wraps a decimal as a Money value.
func NewMoney(d decimal.Decimal) Money { return Money{Decimal: d} }

// MarshalJSON emits a bare JSON number with two decimal places.
func (m Money) MarshalJSON() ([]byte, error) {
	return []byte(m.Decimal.StringFixed(2)), nil
}

// UnmarshalJSON accepts a JSON number or a quoted string.
func (m *Money) UnmarshalJSON(data []byte) error {
	s := strings.TrimSpace(strings.Trim(string(data), `"`))
	if s == "" || s == "null" {
		m.Decimal = decimal.Decimal{}
		return nil
	}
	d, err := decimal.NewFromString(s)
	if err != nil {
		return err
	}
	m.Decimal = d
	return nil
}

// Value implements driver.Valuer for database writes.
func (m Money) Value() (driver.Value, error) { return m.Decimal.Value() }

// Scan implements sql.Scanner for database reads.
func (m *Money) Scan(src any) error { return m.Decimal.Scan(src) }

// MinPayPrice is the lowest payable amount (a minimum of one cent) used by every
// discount strategy when a calculation would otherwise drop to zero or below.
var MinPayPrice = decimal.RequireFromString("0.01")

// ParseMoney parses a decimal amount string, returning an error on malformed input.
func ParseMoney(s string) (decimal.Decimal, error) {
	return decimal.NewFromString(s)
}

// MustMoney parses a decimal amount string and panics on malformed input. It is
// intended for constants and trusted configuration only.
func MustMoney(s string) decimal.Decimal {
	return decimal.RequireFromString(s)
}
