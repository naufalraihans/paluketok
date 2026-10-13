# Urutan Kerja - PaluKetok

Status: **FINAL dan BEKU**. Ditetapkan 2026-10-12. Fase 0 menunggu akses VPS (diberikan user menyusul); fase 1 bisa mulai di laptop. Urutan ini diturunkan dari aturan dependency (`architecture.md` bagian 2), checklist rilis (`security.md` bagian 13), dan checklist Midtrans (`auth_billing.md` bagian 5). Tiap langkah punya tanda selesai yang bisa dicek.

Aturan: satu langkah = satu hasil yang bisa diuji. Langkah berikutnya tidak mulai sebelum tanda selesai terpenuhi.

## Fase 0: Persiapan (tanpa kode)

1. **VPS siap.** SSH key, login password dimatikan, `ufw` buka 22/80/443, `systemd` aktif.
   - Selesai: `ssh` pakai key berhasil, `ufw status` sesuai.
2. **Secret disiapkan.** `server/.env` dibuat dari `.env.example`, nilai diisi lokal, bukan di git.
   - Selesai: `.env` ada di `.gitignore`, `git status` tidak menampilkannya.
3. **DOKU sandbox.** Ikuti checklist `auth_billing.md` bagian 5.
   - Selesai: `DOKU_CLIENT_ID` dan `DOKU_SECRET_KEY` ada, base URL sandbox, Notification URL terdaftar di Back Office.
4. **Supabase.** Project aktif, RLS dinyalakan, email konfirmasi aktif (`auth_billing.md` 1.3).
   - Selesai: bisa daftar lewat Supabase Auth, dan `auth.users` terisi.

## Fase 1: Kerangka server

5. **`go.mod`** dengan module `paluketok/server`, Go 1.26.
6. **`pkg/`**: `config` (viper + env), `response` (format sukses dan error di `openapi.md`), `logger`, `database` (pgxpool).
   - Selesai: `go build ./...` lolos, `go vet` bersih.
7. **`cmd/web/main.go`** dengan composition root kosong. Hanya health check `GET /healthz`.
   - Selesai: `curl localhost:<port>/healthz` balas 200.

## Fase 2: Database

8. **Migrasi `0001_init.sql`** dari `erd.dbml`. Tabel, index, CHECK, unique partial index. Lalu RLS ON dan trigger `profiles` (`auth_billing.md` 1.2).
   - Selesai: migrasi jalan bersih di Supabase sandbox, tidak ada error. Dicek dengan `\d` per tabel.
9. **Uji RLS.** Dua user uji, user A tidak bisa baca data user B.
   - Selesai: query A atas data B kosong, sesuai `security.md` 13.
10. **Uji trigger `profiles`.** Insert `auth.users` membuat satu baris `profiles` dengan `credits = 0`.

Catatan: user uji hanya di sandbox dev. Tidak boleh di prod tanpa izin (lihat aturan user).

## Fase 3: Domain dan usecase (tanpa HTTP, tanpa DB)

11. **`internal/entity`**: struct domain murni. Profile, Project, PlanRun, Artifact, Decision, Order, Ledger.
12. **`internal/port`**: interface repository, LLM, judge, payment, clock.
13. **`internal/usecase`**: logika per fitur. Owner check di sini. Dites dengan fake dari `port`.
    - Selesai: `go list -deps ./internal/usecase/...` tidak memuat `pgx`, `gorm`, `net/http`, `delivery` (`architecture.md` 2). Test unit lolos.

Urutan usecase dalam fase ini:
- 13a. Projects (CRUD owner-scoped)
- 13b. Kredit (potong, refund, idempoten, `FOR UPDATE` lewat port)
- 13c. Runs (buat run, cek satu `running` per project)
- 13d. Orders (buat order, proses webhook, expire)

## Fase 4: Adapter

14. **`repository`**: implementasi port dengan pgx. Transaksi kredit `FOR UPDATE`.
    - Selesai: test integrasi ke Supabase sandbox untuk potong dan refund. Saldo tidak negatif.
15. **`adapter/doku`**: create order (timeout 20 detik), status check (tanpa retry), verifikasi signature HMAC-SHA256 dengan `hmac.Equal`.
    - Selesai: test signature salah ditolak. Test pakai transaksi sandbox sampai `settlement`.
16. **`adapter/llm`**: client OpenAI-compatible, timeout 180 detik. Dites dengan server palsu lokal, bukan key sungguhan.
17. **`adapter/judge`**: client JEV, timeout 30 detik, tanpa retry. Dites mock dulu. Panggilan live berbayar **hanya setelah izin user**.

## Fase 5: HTTP

18. **`delivery/dto`** dan **`delivery/handler`**: parse, panggil usecase, balas via `pkg/response`.
19. **`delivery/middleware`** berurutan: `recover`, `cors`, `bodylimit`, `jwt` (JWKS, `security.md` 2), `admin`, `ratelimit`.
    - Selesai: request tanpa token dapat `401`, token salah `401`, user biasa ke route admin `403`.
20. **`delivery/route`** dan **`cmd/web`** dirangkai. Semua endpoint di `openapi.md` terdaftar.
    - Selesai: tiap endpoint di `openapi.md` punya minimal satu test HTTP.
21. **Webhook DOKU** (`POST /v1/webhooks/doku`). Urutan langkah 1 sampai 8 di `security.md` 6, cabang `order.status` untuk EXPIRED (SEC-D1).
    - Selesai: signature salah = `401`, nominal beda = tolak, sudah settlement = `200` tanpa grant ulang.

## Fase 6: Worker dan planner

22. **`internal/worker`**: pool 4, antrean 20, `503` saat penuh (`security.md` 8, 11).
23. **Pipeline 6 tahap** (`planner.md` 1 sampai 5). Urutan tetap: prd, erd, openapi, roles, security, structure.
    - Selesai per tahap: validasi format lolos (`planner.md` 5). Gagal dua kali = `failed` dan refund.
24. **Gate JEV** (`planner.md` 4). Dites dengan mock. Panggilan live hanya setelah izin.
25. **Restart.** Run `queued` diulang, run `running` ditandai `failed` dengan `server_restart` dan refund (`security.md` 11).
    - Selesai: matikan proses saat run berjalan, nyalakan lagi, cek hasilnya.

## Fase 7: Admin dan rilis

26. **Endpoint admin** (`openapi.md` bagian admin) dan `cmd/seeder` untuk bootstrap admin lewat argumen email.
27. **Deploy ke VPS.** Build di laptop: `GOOS=linux GOARCH=amd64 go build -o paluketok ./cmd/web` (sesuaikan arsitektur VPS). Kirim via `scp`, jalankan lewat `systemd`.
28. **Checklist rilis** `security.md` bagian 13. Setiap item dicentang dengan bukti.
29. **Uji E2E**: daftar user, buat project, run planner sampai `done`, top-up sandbox, regenerate satu tahap.

## Fase 8: Bersih-bersih (pasca BE)

30. Koreksi jejak: `changelog.md` entri fase, `README.md` status akurat, sapu rujukan mati (`tech_stack.md`).
31. `docs/cleanup-candidates.md`: daftar kandidat hapus, belum dihapus. Master yang putuskan.
32. `gaps.md`: blok "Sisa sebelum deploy".

## Fase 9: Frontend Svelte

Stack: **SvelteKit + Go**. FE di `web/`, BE tetap `server/`. Bukan Next/React, bukan framework lain.

33. **Scaffold**: `web/` SvelteKit + Tailwind. Semua warna, font, spacing, radius, copy diambil dari `design/web/` (template beku). Aturan pasti: kelas utility mockup dipakai apa adanya; yang berulang dijadikan token di `app.css` (`@theme`). Dilarang mengubah desain, menambah shadow, menambah emoji, atau mengarang komponen yang tidak ada di mockup.
34. **Tiga halaman** sesuai mockup, satu per satu:
    - `landingPage.html` → `/`
    - `dasrhboard.html` → `/projects` (daftar proyek)
    - `ringkasanProyek.html` → `/projects/[id]` (ringkasan proyek)
    Ukuran, susunan, dan isi tiap halaman harus bisa dibandingkan berdampingan dengan mockup dan tidak berbeda selain data.
35. **Data mock dulu**: tipe + data palsu di `web/src/lib/mock/`. Belum panggil API. Tampilan harus persis template saat data palsu.
36. **Sambung API** (`/v1/*`, Bearer JWT Supabase): ganti mock dengan klien API nyata, satu halaman per langkah. Error state wajib (401 → login, 402 kredit kurang, 422 validasi, 503 antrean penuh).
37. **Verifikasi**: `npm run build` hijau, `npm run check` bersih, tiap halaman dibandingkan screenshot dengan mockup. Lapor bukti, bukan klaim.

Aturan Fase 9:

- `design/web/` beku. Jangan diubah, jangan dihapus.
- Favicon dari `design/favicon.svg`.
- Hover 0.2s smooth, kontras jelas, tanpa emoji di teks AI.
- Jangan tambah dependency UI (komponen sendiri + Tailwind).
- Commit hanya dengan izin Master.

## Di luar urutan ini

- Nama model, base URL, key LLM (diisi admin di panel).
- Tahap `seed`, `test-plan`, `ci` (ditunda).

## Catatan

- Fase 4 dan 6 dengan panggilan live (JEV, LLM) hanya setelah izin user, karena berbiaya.
- Fase 2 dan 4 menyentuh Supabase. Uji di sandbox, bukan prod.
