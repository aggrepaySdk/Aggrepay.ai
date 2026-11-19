package aggrepay

import "testing"

func TestCreateInvoiceParamsValidate(t *testing.T) {
	err := CreateInvoiceParams{Amount: 0, Currency: CurrencyRUB, OrderID: "x"}.validate()
	if err == nil {
		t.Fatal("amount=0 must fail")
	}
	err = CreateInvoiceParams{Amount: 100, Currency: "BTC", OrderID: "x"}.validate()
	if err == nil {
		t.Fatal("bad currency must fail")
	}
	err = CreateInvoiceParams{Amount: 100, Currency: CurrencyRUB, OrderID: "  "}.validate()
	if err == nil {
		t.Fatal("empty order_id must fail")
	}
	err = CreateInvoiceParams{Amount: 100, Currency: CurrencyRUB, OrderID: "order-1"}.validate()
	if err != nil {
		t.Fatal(err)
	}
}

func TestNewClientRequiresKeys(t *testing.T) {
	if _, err := NewClient("", "s"); err == nil {
		t.Fatal("empty apiKey must fail")
	}
	if _, err := NewClient("k", ""); err == nil {
		t.Fatal("empty secret must fail")
	}
}

func TestPaymentStatusTerminal(t *testing.T) {
	if !StatusSuccess.Terminal() {
		t.Fatal("success is terminal")
	}
	if StatusPending.Terminal() {
		t.Fatal("pending is not terminal")
	}
}
