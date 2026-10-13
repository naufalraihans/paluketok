// Validasi format output per tahap (plan/planner.md bagian 2 dan 5):
// 1) kosong/panjang, 2) format (parse DBML/JSON/section markdown),
// 3) konsistensi lintas artefak. Pesan error dipakai ulang untuk retry prompt.
package worker

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"paluketok/server/internal/entity"
)

// reTable dan reRef menangkap subset DBML yang divalidasi.
var (
	reTable = regexp.MustCompile(`(?im)^\s*Table\s+([A-Za-z_][A-Za-z0-9_"]*)`)
	reRef   = regexp.MustCompile(`(?m)\bRef:\s*([A-Za-z_][A-Za-z0-9_."]*)\s*[<>-]+\s*([A-Za-z_][A-Za-z0-9_."]*)`)
)

// maxArtifactChars batas lunak artefak (~64 KB teks). Nilai token asli
// diatur ai_settings (admin); tool ini cegegah output liar.
const maxArtifactChars = 64 << 10

// validate memeriksa draf satu tahap terhadap artefak sebelumnya.
// prev berisi artefak yang sudah lolos (kind -> content).
func validate(stage entity.Stage, draft string, prev map[entity.Stage]string) error {
	if strings.TrimSpace(draft) == "" {
		return fmt.Errorf("artefak %s kosong", stage)
	}
	if len(draft) > maxArtifactChars {
		return fmt.Errorf("artefak %s melebihi %d karakter", stage, maxArtifactChars)
	}
	switch stage {
	case entity.StagePRD:
		return validatePRD(draft)
	case entity.StageERD:
		return validateERD(draft)
	case entity.StageOpenAPI:
		return validateOpenAPI(draft)
	case entity.StageRoles:
		return validateRoles(draft, prev[entity.StageOpenAPI])
	case entity.StageSecurity:
		return validateSecurity(draft)
	case entity.StageStructure:
		return validateStructure(draft, prev)
	}
	return fmt.Errorf("tahap tidak dikenal: %s", stage)
}

// validatePRD: seksi wajib planner.md bagian 2 baris 1.
func validatePRD(d string) error {
	for _, s := range []string{"Tujuan", "Scope", "Acceptance criteria", "Di luar scope"} {
		if !strings.Contains(d, s) {
			return fmt.Errorf("prd tanpa seksi %q", s)
		}
	}
	return nil
}

// validateERD: DBML sederhana - tabel ber-PK, Ref menunjuk tabel yang ada.
// ponytail: parser subset DBML (Table, Ref, pk); penuh bila perlu.
func validateERD(d string) error {
	matches := reTable.FindAllStringSubmatch(d, -1)
	if len(matches) == 0 {
		return fmt.Errorf("erd tidak memuat Table")
	}
	blocks := reTable.FindAllStringSubmatchIndex(d, -1)
	for i, loc := range blocks {
		end := len(d)
		if i+1 < len(blocks) {
			end = blocks[i+1][0]
		}
		body := d[loc[0]:end]
		name := strings.Trim(matches[i][1], `"`)
		if !strings.Contains(body, "[pk]") && !strings.Contains(body, "[primary key]") {
			return fmt.Errorf("tabel %s tanpa primary key", name)
		}
	}
	tables := map[string]bool{}
	for _, m := range matches {
		tables[strings.Trim(m[1], `"`)] = true
	}
	for _, m := range reRef.FindAllStringSubmatch(d, -1) {
		for _, side := range []string{m[1], m[2]} {
			t := strings.Trim(strings.Split(side, ".")[0], ` "`)
			if !tables[t] {
				return fmt.Errorf("ref menunjuk tabel tidak ada: %s", t)
			}
		}
	}
	return nil
}

// validateOpenAPI: JSON valid, pathsNONempty, tiap operasi punya responses.
func validateOpenAPI(d string) error {
	var doc struct {
		OpenAPI string `json:"openapi"`
		Paths   map[string]map[string]struct {
			Responses map[string]json.RawMessage `json:"responses"`
		} `json:"paths"`
	}
	if err := json.Unmarshal([]byte(d), &doc); err != nil {
		return fmt.Errorf("openapi bukan JSON valid: %w", err)
	}
	if doc.OpenAPI == "" {
		return fmt.Errorf("openapi tanpa field openapi")
	}
	if len(doc.Paths) == 0 {
		return fmt.Errorf("openapi tanpa paths")
	}
	for path, ops := range doc.Paths {
		if !strings.HasPrefix(path, "/") {
			return fmt.Errorf("path tidak valid: %s", path)
		}
		for method, op := range ops {
			switch strings.ToLower(method) {
			case "get", "post", "put", "patch", "delete", "head", "options", "trace":
			default:
				return fmt.Errorf("method tidak valid %s %s", method, path)
			}
			if len(op.Responses) == 0 {
				return fmt.Errorf("operasi %s %s tanpa responses", method, path)
			}
		}
	}
	return nil
}

// validateRoles: setiap path openapi muncul di matriks roles.
func validateRoles(d, openapi string) error {
	var doc struct {
		Paths map[string]json.RawMessage `json:"paths"`
	}
	if err := json.Unmarshal([]byte(openapi), &doc); err != nil {
		return fmt.Errorf("openapi tidak bisa dibaca untuk roles: %w", err)
	}
	for path := range doc.Paths {
		if !strings.Contains(d, path) {
			return fmt.Errorf("roles tidak mencakup path %s", path)
		}
	}
	return nil
}

// validateSecurity: sekberisi trust boundary, validasi input, checklist.
func validateSecurity(d string) error {
	for _, s := range []string{"trust boundary", "validasi input", "checklist"} {
		if !strings.Contains(strings.ToLower(d), s) {
			return fmt.Errorf("security tanpa bagian %q", s)
		}
	}
	return nil
}

// validateStructure: tree memuat folder dari stack (fe/be). Info stack
// diteruskan memakai kunci non-stage di prev bila ada.
func validateStructure(d string, prev map[entity.Stage]string) error {
	for _, key := range []string{"stack:fe", "stack:be"} {
		if dir := prev[entity.Stage(key)]; dir != "" && !strings.Contains(d, dir) {
			return fmt.Errorf("structure tanpa folder %s dari stack", strings.TrimPrefix(key, "stack:"))
		}
	}
	return nil
}
