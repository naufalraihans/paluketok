-- ============================================================
-- 0007: plan_stages - skema tahap per run, disusun AI
-- ============================================================
-- Alasan: jumlah dan nama tahap planning dulu dipaku di kode (enam tahap:
-- prd, erd, openapi, roles, security, structure). Akibatnya proyek kecil dan
-- proyek besar dipaksa melewati tahap yang sama. Sekarang skema tahap disusun
-- AI judge dari brief + stack proyek, lalu pengguna boleh mengubahnya sebelum
-- run dimulai.
--
-- Aturan:
--   * FK ke plan_runs ON DELETE CASCADE: skema tidak berguna tanpa run-nya.
--   * key unik per run; label bebas (ditampilkan di UI apa adanya).
--   * position 0-based dan menentukan urutan tahap.
--   * intent = tugas tahap ini dalam bahasa manusia; jadi inti prompt LLM.
--   * output_format menentukan validator yang dipakai (markdown/json/dbml).
--   * artifacts.kind dan plan_runs.current_stage tetap TEXT tanpa FK ke
--     plan_stages: artefak hasil regenerate harus tetap sah walau skema run
--     asalnya sudah tidak ada. Run lama (sebelum migrasi ini) juga tidak punya
--     baris plan_stages sama sekali, dan itu harus tetap bisa dibuka.
--
-- Batas 2..12 tahap ditegakkan di kode (worker/stages.go), bukan CHECK, supaya
-- himpunan format bisa tumbuh tanpa migrasi baru.

create table if not exists public.plan_stages (
  id            uuid primary key default gen_random_uuid(),
  run_id        uuid not null references public.plan_runs(id) on delete cascade,
  key           text not null,
  label         text not null,
  position      integer not null,
  intent        text not null default '',
  output_format text not null default 'markdown',
  check_spec    jsonb,
  depends_on    text[] not null default '{}',
  created_at    timestamptz not null default now(),
  constraint plan_stages_key_shape check (key ~ '^[a-z][a-z0-9_-]{1,30}$'),
  constraint plan_stages_position_nonneg check (position >= 0)
);

comment on table public.plan_stages is
  'Skema tahap planning per run: jumlah, nama, urutan, dan kontrak outputnya. Disusun AI judge sebelum run dibuat.';
comment on column public.plan_stages.output_format is
  'markdown | json | dbml - menentukan validator yang dipakai worker.';
comment on column public.plan_stages.check_spec is
  'Bentuk pemeriksaan tambahan dari judge, mis. {"wajib_seksi": ["Tujuan"]}.';

-- Urutan dibaca selalu per run.
create index if not exists ix_plan_stages_run_position
  on public.plan_stages (run_id, position);

-- Satu key hanya boleh muncul sekali dalam satu run.
create unique index if not exists uq_plan_stages_run_key
  on public.plan_stages (run_id, key);

-- Tidak ada grant ke anon/authenticated: tabel ini hanya diakses server lewat
-- koneksi pemilik database (pola yang sama dengan credits_ledger/ai_settings di
-- 0002_grants_and_guard.sql). RLS aktif bawaan -> klien tidak bisa membaca skema
-- tahap proyek orang lain.
