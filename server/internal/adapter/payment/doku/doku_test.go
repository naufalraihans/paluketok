package doku

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"strings"
	"testing"
	"time"

	"paluketok/server/internal/port"
)

const (
	testSecret   = "SK-test-123"
	testClientID = "BRN-001"
	testPath     = "/v1/webhooks/doku"
)

func expectedSignature(secret, component string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(component))
	return "HMACSHA256=" + base64.StdEncoding.EncodeToString(mac.Sum(nil))
}

// component dokumen-format: "Nama:nilai" per baris.
func docsComponent(clientID, requestID, ts, target string, raw []byte) string {
	s := "Client-Id:" + clientID +
		"\nRequest-Id:" + requestID +
		"\nRequest-Timestamp:" + ts +
		"\nRequest-Target:" + target
	if raw != nil {
		s += "\nDigest:" + base64.StdEncoding.EncodeToString(hashSHA256(raw))
	}
	return s
}

func TestVerify_SignatureBenarLolos(t *testing.T) {
	c := New("https://api-sandbox.doku.com", testClientID, testSecret, testPath)
	raw := []byte(`{"order":{"amount":25000,"invoice_number":"PK-x"}}`)

	ts := "2026-10-14T12:00:00Z"
	sig := expectedSignature(testSecret,
		docsComponent(testClientID, "req-uuid-123", ts, testPath, raw))

	in := port.WebhookInput{
		ClientID: testClientID, RequestID: "req-uuid-123", RequestTimestamp: ts,
		Signature: sig, RequestTarget: testPath, RawBody: raw,
	}
	if err := c.Verify(in); err != nil {
		t.Fatalf("signature benar harus lolos: %v", err)
	}
}

// Verify di webhook memakai helper yang sama: kalau komponen salah, tolak.
func TestVerify_ComponenTanpaNamaDitolak(t *testing.T) {
	c := New("https://api-sandbox.doku.com", testClientID, testSecret, testPath)
	raw := []byte(`{"order":{"amount":25000,"invoice_number":"PK-x"}}`)
	ts := "2026-10-14T12:00:00Z"

	// signature dihitung dari gabungan NILAI polos (format bug lama)
	legacy := testClientID + "\nreq-uuid-123\n" + ts + "\n" + testPath + "\n" +
		base64.StdEncoding.EncodeToString(hashSHA256(raw))
	if err := c.Verify(port.WebhookInput{
		ClientID: testClientID, RequestID: "req-uuid-123", RequestTimestamp: ts,
		Signature:     expectedSignature(testSecret, legacy),
		RequestTarget: testPath, RawBody: raw,
	}); err == nil {
		t.Fatal("format nilai-polos (bug lama) tidak boleh lolos")
	}
}

func TestVerify_SignatureSalahDitolak(t *testing.T) {
	c := New("https://api-sandbox.doku.com", testClientID, testSecret, testPath)
	raw := []byte(`{"order":{"amount":25000,"invoice_number":"PK-x"}}`)
	ts := "2026-10-14T12:00:00Z"

	// signature dihitung dari body BERBEDA (digest tidak cocok)
	other := []byte(`{"order":{"amount":99999,"invoice_number":"PK-x"}}`)
	sig := expectedSignature(testSecret, docsComponent(testClientID, "req-uuid-123", ts, testPath, other))

	err := c.Verify(port.WebhookInput{
		ClientID: testClientID, RequestID: "req-uuid-123", RequestTimestamp: ts,
		Signature: sig, RequestTarget: testPath, RawBody: raw,
	})
	if err != port.ErrForbidden {
		t.Fatalf("signature salah harus forbidden, dapat %v", err)
	}
}

func TestVerify_ClientIdAsingDitolak(t *testing.T) {
	c := New("https://api-sandbox.doku.com", testClientID, testSecret, testPath)
	err := c.Verify(port.WebhookInput{ClientID: "BRN-lain"})
	if err != port.ErrForbidden {
		t.Fatalf("client-id asing harus ditolak, dapat %v", err)
	}
}

// Digest wajib dari body MENTAH: satu spasi berubah = tolak.
func TestVerify_DigestBodyMentahSensitif(t *testing.T) {
	c := New("https://api-sandbox.doku.com", testClientID, testSecret, testPath)
	raw := []byte(`{"order":{"amount":25000}}`)
	ts := "2026-10-14T12:00:00Z"
	sig := expectedSignature(testSecret,
		docsComponent(testClientID, "r1", ts, testPath, raw))

	in := port.WebhookInput{ClientID: testClientID, RequestID: "r1", RequestTimestamp: ts,
		Signature: sig, RequestTarget: testPath}
	in.RawBody = []byte("{ \"order\": {\"amount\": 25000} }") // indent: digest berubah
	if err := c.Verify(in); err == nil {
		t.Fatal("digest dari body indent tidak boleh lolos")
	}
}

// POST (ada body): tepat 5 baris berlabel, tanpa \n akhir.
func TestSignatureComponents_POSTLimaBarisBerlabel(t *testing.T) {
	comp := signatureComponents("CID", "RID", "TS", "/p", []byte("body"))
	n := strings.Count(comp, "\n")
	if n != 4 || strings.HasSuffix(comp, "\n") {
		t.Fatalf("komponen POST harus 5 baris (4 pemisah, tanpa \\n akhir): %q", comp)
	}
	for _, want := range []string{"Client-Id:CID", "Request-Id:RID",
		"Request-Timestamp:TS", "Request-Target:/p", "Digest:"} {
		if !strings.Contains(comp, want) {
			t.Fatalf("baris %q hilang: %q", want, comp)
		}
	}
}

// GET tanpa body: 4 baris, TANPA baris Digest (docs GET method).
func TestSignatureComponents_GETEmpatBarisTanpaDigest(t *testing.T) {
	comp := signatureComponents("CID", "RID", "TS", "/orders/v1/status/PK-1", nil)
	if strings.Count(comp, "\n") != 3 || strings.Contains(comp, "Digest") {
		t.Fatalf("komponen GET harus 4 baris tanpa Digest: %q", comp)
	}
}

func TestParse_Notifikasi(t *testing.T) {
	c := New("", "", "", testPath)
	raw := []byte(`{
	  "order": {"amount": 25000, "invoice_number": "PK-abc", "request_id": "rid-1", "status": "ORDER_EXPIRED"},
	  "transaction": {"status": "SUCCESS"}
	}`)
	n, err := c.Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	if n.InvoiceNumber != "PK-abc" || n.AmountIDR != 25000 || n.RequestID != "rid-1" ||
		n.Transaction != "SUCCESS" || n.OrderStatus != "ORDER_EXPIRED" {
		t.Fatalf("parse salah: %+v", n)
	}
}

// yyyyMMddHHmmss zona UTC+7.
func TestComputeExpired_FormatDanZona(t *testing.T) {
	c := New("", "", "", "")
	base := time.Date(2026, 10, 14, 3, 30, 0, 0, time.UTC) // 10:30 WIB
	got := c.computeExpired(base)
	if got != "20261014103000" {
		t.Fatalf("expired harus 20261014103000 (WIB), dapat %s", got)
	}
}

// Verify dengan body nil: komponen 4 baris tanpa Digest juga berlaku di
// jalur verifikasi (Request-Target diambil dari konfigurasi, bukan header —
// security.md bagian 6).
func TestVerify_TargetGetTanpaBodyEmpatBaris(t *testing.T) {
	c := New("https://api-sandbox.doku.com", testClientID, testSecret, testPath)
	ts := "2026-10-14T12:00:00Z"
	sig := expectedSignature(testSecret,
		docsComponent(testClientID, "rid-get", ts, testPath, nil))
	err := c.Verify(port.WebhookInput{
		ClientID: testClientID, RequestID: "rid-get", RequestTimestamp: ts,
		Signature: sig, RequestTarget: testPath,
	})
	if err != nil {
		t.Fatalf("verifikasi 4 baris tanpa body harus lolos: %v", err)
	}
}
