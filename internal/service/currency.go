package service

import (
	"errors"
	"fmt"
)

// Supported Currency String Literals
const (
	CurrencyUSD = "USD"
	CurrencyINR = "INR"
)

var ErrUnsupportedCurrency = errors.New("unsupported currency asset type")

type CurrencyService struct {
	// Mock static exchange rate factor: 1 USD = 83.50 INR
	// We scale the factor to maintain pure integer arithmetic bounds
	usdToInrRateFloat float64
}

func NewCurrencyService() *CurrencyService {
	return &CurrencyService{
		usdToInrRateFloat: 83.50,
	}
}

// ConvertMinorUnits handles conversion conversions natively using absolute integer precision
func (s *CurrencyService) ConvertMinorUnits(amountMinor int64, fromCurrency, toCurrency string) (int64, error) {
	if fromCurrency == toCurrency {
		return amountMinor, nil
	}

	switch {
	case fromCurrency == CurrencyUSD && toCurrency == CurrencyINR:
		// Example: 100 cents ($1.00) * 83.50 = 8350 paise (₹83.50)
		converted := float64(amountMinor) * s.usdToInrRateFloat
		return int64(converted), nil

	case fromCurrency == CurrencyINR && toCurrency == CurrencyUSD:
		// Example: 8350 paise (₹83.50) / 83.50 = 100 cents ($1.00)
		converted := float64(amountMinor) / s.usdToInrRateFloat
		return int64(converted), nil

	default:
		return 0, fmt.Errorf("%w: cannot convert from %s to %s", ErrUnsupportedCurrency, fromCurrency, toCurrency)
	}
}

// FormatToMajorUnits prints the minor integer out as a human-readable standard currency float presentation
func FormatToMajorUnits(amountMinor int64) string {
	major := float64(amountMinor) / 100.0
	return fmt.Sprintf("%.2f", major)
}
