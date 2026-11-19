package aggrepay

import "strings"

// Статусы инвойса по контракту Merchant API.
type PaymentStatus string

const (
	StatusPending    PaymentStatus = "pending"
	StatusProcessing PaymentStatus = "processing"
	StatusSuccess    PaymentStatus = "success"
	StatusFailed     PaymentStatus = "failed"
	StatusExpired    PaymentStatus = "expired"
	StatusRefunded   PaymentStatus = "refunded"
	StatusChargeback PaymentStatus = "chargeback"
)

func (s PaymentStatus) Terminal() bool {
	switch s {
	case StatusSuccess, StatusFailed, StatusExpired, StatusRefunded, StatusChargeback:
		return true
	default:
		return false
	}
}

type Currency string

const (
	CurrencyRUB Currency = "RUB"
	CurrencyUSD Currency = "USD"
	CurrencyEUR Currency = "EUR"
)

// CreateInvoiceParams: тело POST /api/v1/invoice/create.
// Amount в минимальных единицах (10000 = 100.00 RUB).
type CreateInvoiceParams struct {
	Amount          int64    `json:"amount"`
	Currency        Currency `json:"currency"`
	OrderID         string   `json:"order_id"`
	Description     string   `json:"description,omitempty"`
	ProjectURL      string   `json:"project_url,omitempty"`
	RedirectURL     string   `json:"redirect_url,omitempty"`
	FailRedirectURL string   `json:"fail_redirect_url,omitempty"`
	CallbackURL     string   `json:"callback_url,omitempty"`
}

func (p CreateInvoiceParams) validate() error {
	if p.Amount < 1 || p.Amount > 30_000_000 {
		return &ValidationError{Field: "amount", Message: "must be between 1 and 30000000"}
	}
	switch p.Currency {
	case CurrencyRUB, CurrencyUSD, CurrencyEUR:
	default:
		return &ValidationError{Field: "currency", Message: "must be RUB, USD or EUR"}
	}
	if strings.TrimSpace(p.OrderID) == "" {
		return &ValidationError{Field: "order_id", Message: "required"}
	}
	return nil
}

// InvoiceCreated: data после create.
type InvoiceCreated struct {
	InvoiceID     string        `json:"invoice_id"`
	PayURL        string        `json:"pay_url"`
	PaymentStatus PaymentStatus `json:"payment_status"`
}

// Invoice: GET /api/v1/invoice/{id}.
type Invoice struct {
	InvoiceID          string        `json:"invoice_id"`
	MerchantID         string        `json:"merchant_id"`
	OrderID            string        `json:"order_id"`
	Amount             int64         `json:"amount"`
	Currency           Currency      `json:"currency"`
	Description        string        `json:"description,omitempty"`
	Status             PaymentStatus `json:"status"`
	ProviderID         *string       `json:"provider_id"`
	ProviderAccountID  *string       `json:"provider_account_id"`
	PayURL             string        `json:"pay_url,omitempty"`
	ProjectURL         string        `json:"project_url,omitempty"`
	RedirectURL        string        `json:"redirect_url,omitempty"`
	FailRedirectURL    string        `json:"fail_redirect_url,omitempty"`
	CallbackURL        string        `json:"callback_url,omitempty"`
	ExpiresAt          string        `json:"expires_at"`
	CreatedAt          string        `json:"created_at"`
	UpdatedAt          string        `json:"updated_at"`
	ReferralAmount     *int64        `json:"referral_amount,omitempty"`
	ReferrerMerchantID *string       `json:"referrer_merchant_id"`
}

// HistoryItem: элемент GET .../history.
type HistoryItem struct {
	OldStatus *PaymentStatus `json:"old_status"`
	NewStatus PaymentStatus  `json:"new_status"`
	Source    string         `json:"source"`
	Reason    string         `json:"reason,omitempty"`
	CreatedAt string         `json:"created_at"`
}

// TestSecureResult: data POST /api/v1/merchant/test-secure.
type TestSecureResult struct {
	MerchantID string `json:"merchant_id"`
	Message    string `json:"message"`
}

// CallbackPayload: тело исходящего callback мерчанту.
type CallbackPayload struct {
	InvoiceID         string        `json:"invoice_id"`
	OrderID           string        `json:"order_id"`
	Amount            int64         `json:"amount"`
	Currency          Currency      `json:"currency"`
	Status            PaymentStatus `json:"status"`
	Provider          string        `json:"provider"`
	ProviderPaymentID string        `json:"provider_payment_id,omitempty"`
	Timestamp         string        `json:"timestamp"`
}
