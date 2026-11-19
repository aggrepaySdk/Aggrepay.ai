package aggrepay

import "testing"

func TestRequestCanonicalMatchesDocs(t *testing.T) {
	body := []byte("{}")
	wantSHA := "44136fa355b3678a1146ad16f7e8649e94fb4fc21fe77e8310c060f61caaff8a"
	if got := BodySHA256Hex(body); got != wantSHA {
		t.Fatalf("body sha: got %s want %s", got, wantSHA)
	}
	got := RequestCanonical("POST", "/api/v1/merchant/test-secure", "1716883200", "8d52f2e4f39f4f45a567a3b6e9b4be11", body)
	want := "POST\n/api/v1/merchant/test-secure\n1716883200\n8d52f2e4f39f4f45a567a3b6e9b4be11\n" + wantSHA
	if got != want {
		t.Fatalf("canonical mismatch\n got %q\nwant %q", got, want)
	}
}

func TestVerifyCallbackRejectsEmpty(t *testing.T) {
	if VerifyCallback("", "1", []byte("{}"), "abc") {
		t.Fatal("empty secret must fail")
	}
	if VerifyCallback("secret", "", []byte("{}"), "abc") {
		t.Fatal("empty timestamp must fail")
	}
}

func TestVerifyCallbackRoundtrip(t *testing.T) {
	body := []byte(`{"invoice_id":"inv_1"}`)
	ts := "1700000000"
	sig := Sign("test-secret", CallbackCanonical(ts, body))
	if !VerifyCallback("test-secret", ts, body, sig) {
		t.Fatal("expected valid signature")
	}
	if VerifyCallback("test-secret", ts, body, "deadbeef") {
		t.Fatal("expected invalid signature")
	}
}
