// E2E langkah 29 (workflow.md): alur penuh via HTTP asli ke server httptest:
// login-JWT -> project -> run sampai done (LLM/JEV mock) -> artifact ->
// webhook Midtrans tervalidasi signature -> regenerate satu tahap. Tanpa
// service eksternal (Get Status API ditiru fake TransactionVerifier).
package httpapi

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

// req bantu: request HTTP dengan bearer JWT.
func (e *e2eEnv) do(t *testing.T, method, path, tok string, body any) (int, map[string]any) {
	t.Helper()
	var rdr *bytes.Reader
	if body != nil {
		raw, _ := json.Marshal(body)
		rdr = bytes.NewReader(raw)
	} else {
		rdr = bytes.NewReader(nil)
	}
	req, err := http.NewRequest(method, e.srv.URL+path, rdr)
	if err != nil {
		t.Fatalf("%s %s: %v", method, path, err)
	}
	if tok != "" {
		req.Header.Set("Authorization", "Bearer "+tok)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("%s %s: %v", method, path, err)
	}
	defer resp.Body.Close()
	var out map[string]any
	_ = json.NewDecoder(resp.Body).Decode(&out)
	return resp.StatusCode, out
}

// waitFor menunggu kondisi (poll status run) sampai deadline (hook worker async).
func waitFor(t *testing.T, timeout time.Duration, cond func() bool, what string) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("timeout menunggu %s", what)
}

func TestE2E_FullFlow(t *testing.T) {
	e := newE2E(t)
	tok := e.fix.token(t, uidE2E)

	// 1. GET /v1/me — JWT valid -> 200.
	code, me := e.do(t, "GET", "/v1/me", tok, nil)
	if code != 200 {
		t.Fatalf("me: got %d %v", code, me)
	}

	// 2. POST /v1/projects.
	code, proj := e.do(t, "POST", "/v1/projects", tok, map[string]any{
		"name": "E2E", "brief": "Aplikasi kasir sederhana", "stack": map[string]string{
			"fe": "svelte", "be": "go", "db": "postgres"}})
	if code != 201 {
		t.Fatalf("create project: got %d %v", code, proj)
	}
	pid := proj["id"].(string)

	// 3. POST /v1/projects/{id}/runs -> 202, run queued, kredit terpotong 1.
	code, run := e.do(t, "POST", "/v1/projects/"+pid+"/runs", tok, nil)
	if code != 202 && code != 201 {
		t.Fatalf("create run: got %d %v", code, run)
	}
	rid := run["id"].(string)
	if run["status"] != "queued" {
		t.Fatalf("run status: %v", run["status"])
	}

	// 4. Tunggu run done (worker pool asli, LLM/JEV mock).
	waitFor(t, 5*time.Second, func() bool {
		c, r := e.do(t, "GET", "/v1/runs/"+rid, tok, nil)
		return c == 200 && r["status"] == "done"
	}, "run done")
	_ = rid
	_ = strings.TrimSpace

	// 5. GET artifacts run -> 6 tahap final.
	code, arts := e.do(t, "GET", "/v1/runs/"+rid+"/artifacts", tok, nil)
	if code != 200 {
		t.Fatalf("artifacts: %d", code)
	}
	list := arts["artifacts"].([]any)
	if len(list) < 6 {
		t.Fatalf("artifacts kurang: %d", len(list))
	}

	// 6. Webhook Midtrans: top-up 25.000 -> settlement.
	// Order dibuat dulu (POST /v1/orders). Midtrans mengirim signature_key
	// yang harus cocok dengan SHA512(order_id+status_code+gross_amount+
	// ServerKey); di sini ServerKey dipegang memPayment (e2eServerKey). Setelah
	// itu status terkini dibaca ulang dari memPayment (Get Status API palsu).
	code, ord := e.do(t, "POST", "/v1/orders", tok, map[string]any{"package_id": "starter"})
	if code != 201 {
		t.Fatalf("order: %d %v", code, ord)
	}
	oid := ord["order_id"].(string)
	inv := ord["invoice_number"].(string)

	// Saldo sebelum topup.
	_, me2 := e.do(t, "GET", "/v1/me", tok, nil)
	before := int(me2["credits"].(float64))

	// Transaksi yang "ada di Midtrans": kuncinya order_id = invoice kita.
	e.pay.setTx(inv, 25000, "settlement", true)

	req, _ := http.NewRequest("POST", e.srv.URL+e.notify,
		bytes.NewReader(e2eNotifBody(inv, "25000.00", "settlement")))
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	rb, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("webhook: %d %s", resp.StatusCode, rb)
	}
	// Order settlement + saldo naik.
	_, ord2 := e.do(t, "GET", "/v1/orders/"+oid, tok, nil)
	if ord2["status"] != "settlement" {
		t.Fatalf("order status: %v", ord2["status"])
	}
	_, me3 := e.do(t, "GET", "/v1/me", tok, nil)
	after := int(me3["credits"].(float64))
	if after-before != 10 {
		t.Fatalf("saldo: %d -> %d (harus +10)", before, after)
	}

	// 7. Regenerate satu tahap: POST /v1/artifacts/{id}/regenerate.
	art := list[0].(map[string]any)
	aid := art["id"].(string)
	nCalls := len(e.llm.snapshot())
	code, rg := e.do(t, "POST", "/v1/artifacts/"+aid+"/regenerate", tok, nil)
	if code != 200 && code != 202 {
		t.Fatalf("regenerate: %d %v", code, rg)
	}
	waitFor(t, 5*time.Second, func() bool {
		return len(e.llm.snapshot()) > nCalls
	}, "regenerate call")
	c2, art2 := e.do(t, "GET", "/v1/artifacts/"+aid, tok, nil)
	if c2 != 200 || art2["version"].(float64) < 2 {
		t.Fatalf("artifact versi: %d %v", c2, art2)
	}
}
