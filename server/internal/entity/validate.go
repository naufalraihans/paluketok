package entity

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"unicode"
)

// Batas panjang dari plan/security.md bagian 4 dan plan/erd.dbml.
const (
	MaxNameLen       = 120
	MaxBriefLen      = 8000
	MaxStackValueLen = 32
	MaxPackageIDLen  = 32
)

// uuidRe memvalidasi bentuk UUID sebelum query (plan/security.md bagian 4:
// "run_id, artifact_id, order_id | UUID valid, cek format sebelum query").
var uuidRe = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

// ValidID true bila s berbentuk UUID.
func ValidID(s string) bool { return uuidRe.MatchString(s) }

// ValidPlanTier plan_tier hanya free|pro (erd.dbml profiles.plan_tier).
func ValidPlanTier(t PlanTier) bool { return t == TierFree || t == TierPro }

// ValidBaseURLAllowlist memvalidasi base_url slot AI (security.md tabel :72):
// HTTPS wajib, host non-lokal; daftar host dijaga admin via env koma.
func ValidBaseURLAllowlist(raw string) error {
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" {
		return errors.New("base_url tidak valid")
	}
	if u.Scheme != "https" {
		return errors.New("base_url wajib https")
	}
	host := strings.ToLower(u.Hostname())
	if host == "localhost" || host == "127.0.0.1" || host == "::1" || strings.HasSuffix(host, ".local") {
		return errors.New("base_url host lokal tidak diizinkan")
	}
	return nil
}

// ErrValidation membungkus pesan validasi. Pesan TIDAK boleh membocorkan nama
// tabel atau query (plan/security.md bagian 4).
type ErrValidation struct{ Msg string }

func (e *ErrValidation) Error() string { return e.Msg }

func invalid(format string, args ...any) error {
	return &ErrValidation{Msg: fmt.Sprintf(format, args...)}
}

// NormalizeName memangkas dan memvalidasi projects.name: 1-120 karakter, tanpa
// karakter kontrol (plan/security.md bagian 4).
func NormalizeName(raw string) (string, error) {
	name := strings.TrimSpace(raw)
	if name == "" {
		return "", invalid("nama project wajib diisi")
	}
	if len([]rune(name)) > MaxNameLen {
		return "", invalid("nama project maksimal %d karakter", MaxNameLen)
	}
	for _, r := range name {
		if unicode.IsControl(r) {
			return "", invalid("nama project memuat karakter kontrol")
		}
	}
	return name, nil
}

// ValidateBrief memvalidasi projects.brief: 1-8000 karakter (erd.dbml CHECK).
func ValidateBrief(raw string) error {
	n := len([]rune(raw))
	if n == 0 {
		return invalid("brief wajib diisi")
	}
	if n > MaxBriefLen {
		return invalid("brief maksimal %d karakter", MaxBriefLen)
	}
	return nil
}

// stackKeys adalah key yang diizinkan di projects.stack
// (plan/security.md bagian 4: key hanya fe, be, db, payment).
var stackKeys = map[string]bool{"fe": true, "be": true, "db": true, "payment": true}

// DecodeStack mem-parse projects.stack dengan menolak key tak dikenal
// (plan/security.md bagian 4). Nilai kosong berarti stack tidak diisi.
//
// Catatan: daftar nilai yang diizinkan ("whitelist enum") belum ada di plan/;
// yang ditegakkan di sini hanya nama key dan bentuk nilai. Lihat laporan.
func DecodeStack(raw json.RawMessage) (Stack, error) {
	if len(raw) == 0 {
		return Stack{}, nil
	}
	dec := json.NewDecoder(strings.NewReader(string(raw)))
	dec.DisallowUnknownFields()

	var s Stack
	if err := dec.Decode(&s); err != nil {
		return Stack{}, invalid("stack harus object dengan key fe, be, db, payment")
	}
	// DisallowUnknownFields tidak menangkap key duplikat atau key di luar struct
	// bila nilainya null; cek eksplisit sekali lagi.
	var asMap map[string]json.RawMessage
	if err := json.Unmarshal(raw, &asMap); err != nil {
		return Stack{}, invalid("stack harus object JSON")
	}
	for k := range asMap {
		if !stackKeys[k] {
			return Stack{}, invalid("stack memuat key yang tidak dikenal: %s", k)
		}
	}
	if err := validateStackValues(asMap); err != nil {
		return Stack{}, err
	}
	return s, nil
}

// EncodeStack mengubah Stack menjadi jsonb, atau nil bila kosong.
func EncodeStack(s Stack) (json.RawMessage, error) {
	if s.IsZero() {
		return nil, nil
	}
	b, err := json.Marshal(s)
	if err != nil {
		return nil, fmt.Errorf("encode stack: %w", err)
	}
	return b, nil
}

func validateStackValues(asMap map[string]json.RawMessage) error {
	for k, v := range asMap {
		var val string
		if err := json.Unmarshal(v, &val); err != nil {
			return invalid("stack.%s harus teks", k)
		}
		val = strings.TrimSpace(val)
		if val == "" {
			return invalid("stack.%s tidak boleh kosong", k)
		}
		if len(val) > MaxStackValueLen {
			return invalid("stack.%s maksimal %d karakter", k, MaxStackValueLen)
		}
		for _, r := range val {
			if !unicode.IsLower(r) && !unicode.IsDigit(r) && r != '-' && r != '_' && r != '+' && r != '.' {
				return invalid("stack.%s hanya boleh huruf kecil, angka, dan - _ + .", k)
			}
		}
	}
	return nil
}

// ValidProjectStatus true bila status termasuk enum projects.status.
func ValidProjectStatus(s ProjectStatus) bool {
	switch s {
	case ProjectDraft, ProjectReady, ProjectArchived:
		return true
	}
	return false
}

// ValidStage true bila kind termasuk enam tahap planner.
func ValidStage(s Stage) bool {
	for _, st := range Stages {
		if st == s {
			return true
		}
	}
	return false
}
