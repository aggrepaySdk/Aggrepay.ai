package aggrepay

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	DefaultBaseURL   = "https://aggrepay.ai"
	defaultTimeout   = 30 * time.Second
	maxResponseBytes = 2 << 20 // 2 MiB
	userAgent        = "aggrepay-go/1.0.0"
	maxRetries       = 2
)

type Client struct {
	apiKey     string
	secretKey  string
	baseURL    string
	httpClient *http.Client
	userAgent  string
	maxRetries int
}

type Option func(*Client)

func WithBaseURL(u string) Option {
	return func(c *Client) { c.baseURL = strings.TrimRight(u, "/") }
}

func WithHTTPClient(h *http.Client) Option {
	return func(c *Client) { c.httpClient = h }
}

func WithUserAgent(ua string) Option {
	return func(c *Client) { c.userAgent = ua }
}

func WithMaxRetries(n int) Option {
	return func(c *Client) {
		if n < 0 {
			n = 0
		}
		c.maxRetries = n
	}
}

func NewClient(apiKey, secretKey string, opts ...Option) (*Client, error) {
	if strings.TrimSpace(apiKey) == "" || strings.TrimSpace(secretKey) == "" {
		return nil, &ValidationError{Field: "credentials", Message: "apiKey and secretKey are required"}
	}
	c := &Client{
		apiKey:     apiKey,
		secretKey:  secretKey,
		baseURL:    DefaultBaseURL,
		httpClient: &http.Client{Timeout: defaultTimeout},
		userAgent:  userAgent,
		maxRetries: maxRetries,
	}
	for _, opt := range opts {
		opt(c)
	}
	if err := validateBaseURL(c.baseURL); err != nil {
		return nil, err
	}
	return c, nil
}

func validateBaseURL(raw string) error {
	u, err := url.ParseRequestURI(raw)
	if err != nil {
		return &ValidationError{Field: "baseURL", Message: err.Error()}
	}
	host := u.Hostname()
	local := host == "localhost" || host == "127.0.0.1" || host == "::1"
	if u.Scheme != "https" && !(u.Scheme == "http" && local) {
		return &ValidationError{Field: "baseURL", Message: "only https is allowed (http for localhost)"}
	}
	return nil
}

func (c *Client) TestSecure(ctx context.Context) (*TestSecureResult, error) {
	// Ключ один на вызов: ретраи несут тот же Idempotency-Key.
	key, err := newIdempotencyKey("test-secure")
	if err != nil {
		return nil, &TransportError{Op: "nonce", Err: err}
	}
	var out TestSecureResult
	if err := c.do(ctx, http.MethodPost, "/api/v1/merchant/test-secure", map[string]any{}, key, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) CreateInvoice(ctx context.Context, p CreateInvoiceParams, idempotencyKey string) (*InvoiceCreated, error) {
	if err := p.validate(); err != nil {
		return nil, err
	}
	if idempotencyKey == "" {
		idempotencyKey = "create-" + p.OrderID
	}
	var out InvoiceCreated
	if err := c.do(ctx, http.MethodPost, "/api/v1/invoice/create", p, idempotencyKey, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) GetInvoice(ctx context.Context, invoiceID string) (*Invoice, error) {
	if strings.TrimSpace(invoiceID) == "" {
		return nil, &ValidationError{Field: "invoice_id", Message: "required"}
	}
	var out Invoice
	path := "/api/v1/invoice/" + url.PathEscape(invoiceID)
	if err := c.do(ctx, http.MethodGet, path, nil, "", &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) InvoiceHistory(ctx context.Context, invoiceID string) ([]HistoryItem, error) {
	if strings.TrimSpace(invoiceID) == "" {
		return nil, &ValidationError{Field: "invoice_id", Message: "required"}
	}
	var out []HistoryItem
	path := "/api/v1/invoice/" + url.PathEscape(invoiceID) + "/history"
	if err := c.do(ctx, http.MethodGet, path, nil, "", &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *Client) VerifyCallback(rawBody []byte, timestamp, signature string) bool {
	return VerifyCallback(c.secretKey, timestamp, rawBody, signature)
}

type envelope struct {
	Status string          `json:"status"`
	Data   json.RawMessage `json:"data"`
	Error  *struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
	// Плоский формат: {status,code,message}
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (c *Client) do(ctx context.Context, method, path string, jsonBody any, idempotencyKey string, dest any) error {
	var body []byte
	if jsonBody != nil {
		b, err := json.Marshal(jsonBody)
		if err != nil {
			return &TransportError{Op: "marshal", Err: err}
		}
		body = b
	}

	// idempotencyKey фиксирован до цикла. На GET пустой.
	attempts := 1 + c.maxRetries
	var lastErr error
	for attempt := 0; attempt < attempts; attempt++ {
		if attempt > 0 {
			wait := retryWait(lastErr, attempt)
			select {
			case <-ctx.Done():
				return &TransportError{Op: "retry", Err: ctx.Err()}
			case <-time.After(wait):
			}
		}
		err := c.roundTrip(ctx, method, path, body, idempotencyKey, dest)
		if err == nil {
			return nil
		}
		lastErr = err
		if !retryable(err) {
			return err
		}
	}
	return lastErr
}

func (c *Client) roundTrip(ctx context.Context, method, path string, body []byte, idempotencyKey string, dest any) error {
	ts := fmt.Sprintf("%d", time.Now().Unix())
	n, err := nonce(16)
	if err != nil {
		return &TransportError{Op: "nonce", Err: err}
	}
	canonical := RequestCanonical(method, path, ts, n, body)
	sig := Sign(c.secretKey, canonical)

	var rdr io.Reader
	if len(body) > 0 {
		rdr = bytes.NewReader(body)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, rdr)
	if err != nil {
		return &TransportError{Op: "new_request", Err: err}
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", c.userAgent)
	req.Header.Set("X-API-KEY", c.apiKey)
	req.Header.Set("X-TIMESTAMP", ts)
	req.Header.Set("X-NONCE", n)
	req.Header.Set("X-SIGNATURE", sig)
	if idempotencyKey != "" {
		req.Header.Set("Idempotency-Key", idempotencyKey)
	}
	if len(body) > 0 {
		req.Header.Set("Content-Type", "application/json")
	}

	res, err := c.httpClient.Do(req)
	if err != nil {
		return &TransportError{Op: "do", Err: err}
	}
	defer res.Body.Close()

	retryAfter := parseRetryAfter(res.Header.Get("Retry-After"))

	raw, err := io.ReadAll(io.LimitReader(res.Body, maxResponseBytes+1))
	if err != nil {
		return &TransportError{Op: "read", Err: err, HTTPStatus: res.StatusCode, RetryAfter: retryAfter}
	}
	if len(raw) > maxResponseBytes {
		return &TransportError{
			Op:         "read",
			Err:        errors.New("response body too large"),
			HTTPStatus: res.StatusCode,
			RetryAfter: retryAfter,
		}
	}

	var env envelope
	if err := json.Unmarshal(raw, &env); err != nil {
		snippet := string(raw)
		if len(snippet) > 256 {
			snippet = snippet[:256]
		}
		return &TransportError{
			Op:          "decode",
			Err:         err,
			HTTPStatus:  res.StatusCode,
			BodySnippet: snippet,
			RetryAfter:  retryAfter,
		}
	}

	if res.StatusCode >= 400 || env.Status == "error" {
		msg := "API error"
		code := ""
		if env.Error != nil && (env.Error.Message != "" || env.Error.Code != "") {
			msg = env.Error.Message
			code = env.Error.Code
		} else if env.Message != "" || env.Code != "" {
			msg = env.Message
			code = env.Code
		}
		if msg == "" {
			msg = "API error"
		}
		return &APIError{
			Message:    msg,
			Code:       code,
			HTTPStatus: res.StatusCode,
			Payload:    env,
			RetryAfter: retryAfter,
		}
	}

	if env.Status != "" && env.Status != "success" {
		return &TransportError{
			Op:         "status",
			Err:        fmt.Errorf("unexpected status %q", env.Status),
			HTTPStatus: res.StatusCode,
		}
	}

	if dest == nil {
		return nil
	}
	if len(env.Data) == 0 || string(env.Data) == "null" {
		return &TransportError{
			Op:         "decode_data",
			Err:        errors.New("empty data in success response"),
			HTTPStatus: res.StatusCode,
		}
	}
	if err := json.Unmarshal(env.Data, dest); err != nil {
		return &TransportError{Op: "decode_data", Err: err, HTTPStatus: res.StatusCode}
	}
	return nil
}

func retryable(err error) bool {
	var te *TransportError
	if errors.As(err, &te) {
		if te.HTTPStatus == 0 {
			return true
		}
		return te.HTTPStatus == 429 || te.HTTPStatus >= 500
	}
	var ae *APIError
	if errors.As(err, &ae) {
		return ae.HTTPStatus == 429 || ae.HTTPStatus >= 500
	}
	return false
}

func retryWait(err error, attempt int) time.Duration {
	if d := retryAfterOf(err); d > 0 {
		return d
	}
	d := 200 * time.Millisecond
	for i := 1; i < attempt; i++ {
		d *= 2
	}
	return d
}

func retryAfterOf(err error) time.Duration {
	var te *TransportError
	if errors.As(err, &te) && te.RetryAfter > 0 {
		return te.RetryAfter
	}
	var ae *APIError
	if errors.As(err, &ae) && ae.RetryAfter > 0 {
		return ae.RetryAfter
	}
	return 0
}

func parseRetryAfter(v string) time.Duration {
	v = strings.TrimSpace(v)
	if v == "" {
		return 0
	}
	if sec, err := strconv.Atoi(v); err == nil && sec >= 0 {
		return time.Duration(sec) * time.Second
	}
	if t, err := http.ParseTime(v); err == nil {
		d := time.Until(t)
		if d < 0 {
			return 0
		}
		return d
	}
	return 0
}

func nonce(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func newIdempotencyKey(prefix string) (string, error) {
	n, err := nonce(8)
	if err != nil {
		return "", err
	}
	return prefix + "-" + n, nil
}
