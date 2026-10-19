package entity

import "testing"

// Base URL wajib ditulis LENGKAP sampai endpointnya. Server tidak menambah
// path apa pun, jadi berhenti di "/v1" berarti request menembak alamat yang
// bukan endpoint. Aturan ini lahir dari insiden nyata: slot judge berisi
// ".../v1/systemone" dan tombol tes koneksi selalu merah.
func TestValidBaseURLEndpoint(t *testing.T) {
	sah := []string{
		"https://api.experientiallabs.ai/v1/decisions",
		"https://gw.example.com/v1/chat/completions",
		"https://lucif.ai/v1/chat/completions",
	}
	for _, raw := range sah {
		if err := ValidBaseURLEndpoint(raw); err != nil {
			t.Errorf("harus sah: %q -> %v", raw, err)
		}
	}

	tolak := map[string]string{
		"https://api.example.com/v1":                  "berhenti di /v1",
		"https://api.example.com":                     "tanpa path",
		"https://api.example.com/":                    "path kosong",
		"https://api.example.com/v1/":                 "berhenti di /v1 dengan slash",
		"http://api.example.com/v1/decisions":         "bukan https",
		"https://localhost/v1/decisions":              "host lokal",
		"https://127.0.0.1/v1/decisions":              "host lokal",
		"https://gw.local/v1/decisions":               "host .local",
		"bukan-url":                                   "bukan URL",
		"https://api.example.com/v1/decisions?x=1#y2": "", // query/fragment: tetap sah
	}
	for raw, alasan := range tolak {
		err := ValidBaseURLEndpoint(raw)
		if alasan == "" {
			if err != nil {
				t.Errorf("harus sah (query/fragment): %q -> %v", raw, err)
			}
			continue
		}
		if err == nil {
			t.Errorf("harus ditolak (%s): %q", alasan, raw)
		}
	}
}
