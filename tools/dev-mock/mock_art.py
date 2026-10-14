"""Artefak mock per tahap; memenuhi internal/worker/validate.go.

Dipakai mock_llm.py. Isi statis, deterministik, tanpa panggilan luar.
"""

PRD = """# PRD Mock

## Tujuan
Menyediakan contoh artefak PRD untuk uji flow lokal.

## Scope
- Satu endpoint contoh.
- Satu tabel contoh.

## Acceptance criteria
- Endpoint membalas 200.
- Tabel punya primary key.

## Di luar scope
- Integrasi pihak ketiga.
"""

ERD = """Table users {
  id uuid [pk]
  email text
}

Table items {
  id uuid [pk]
  user_id uuid
}

Ref: items.user_id > users.id
"""

OPENAPI = """{
  "openapi": "3.1.0",
  "info": { "title": "Mock API", "version": "1.0.0" },
  "paths": {
    "/v1/items": {
      "get": { "responses": { "200": { "description": "ok" } } }
    }
  }
}"""

ROLES = """# Matriks Roles

| Path | admin | user |
| --- | --- | --- |
| /v1/items | ya | ya |
"""

SECURITY = """# Security Mock

## Trust boundary
Batas antara klien dan server.

## Validasi input
Semua input divalidasi panjang dan tipe.

## Checklist
- [x] auth diperiksa
- [x] error tidak bocorkan detail
"""

STRUCTURE = """# Struktur Mock

```
api/
  cmd/
  internal/
web/
  src/
```

Aturan layer: handler -> usecase -> repository.
"""

MAP = {
    "prd": PRD,
    "erd": ERD,
    "openapi": OPENAPI,
    "roles": ROLES,
    "security": SECURITY,
    "structure": STRUCTURE,
}


def build(stage: str) -> str:
    """Artefak untuk satu tahap; tak dikenal -> PRD."""
    return MAP.get(stage, PRD)