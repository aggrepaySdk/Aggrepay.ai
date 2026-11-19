package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/aggrepaySdk/Aggrepay.ai/golang/aggrepay"
)

func main() {
	client, err := aggrepay.NewClient(
		env("AGGREPAY_API_KEY", ""),
		env("AGGREPAY_SECRET_KEY", ""),
	)
	must(err)

	ctx := context.Background()

	ping, err := client.TestSecure(ctx)
	must(err)
	fmt.Println("testSecure:", ping.MerchantID, ping.Message)

	inv, err := client.CreateInvoice(ctx, aggrepay.CreateInvoiceParams{
		Amount:          10000,
		Currency:        aggrepay.CurrencyRUB,
		OrderID:         fmt.Sprintf("order-%d", time.Now().Unix()),
		Description:     "Demo order",
		ProjectURL:      "https://shop.example",
		RedirectURL:     "https://shop.example/success",
		FailRedirectURL: "https://shop.example/fail",
		CallbackURL:     "https://shop.example/payment/callback",
	}, "")
	must(err)

	fmt.Println("invoice_id:", inv.InvoiceID)
	fmt.Println("pay_url:", inv.PayURL)
	fmt.Println("status:", inv.PaymentStatus)
}

func env(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func must(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
