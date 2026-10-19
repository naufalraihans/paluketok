-- 0009: skill perancangan.
--
-- Skill = instruksi tersimpan yang disuntik ke system prompt saat dipakai di
-- chat. Gunanya mengunci cara berpikir yang sudah terbukti, sehingga pengguna
-- tidak perlu mengetik ulang aturan yang sama di setiap percakapan.
--
-- Bentuknya FILE MARKDOWN (`isi`), bukan kolom-kolom terpisah. Alasannya:
-- instruksi bagus hampir selalu berisi daftar, contoh, dan pengecualian, dan
-- memaksanya masuk kolom terstruktur membuatnya tidak terbaca.
--
-- Cakupan GLOBAL dan hanya admin yang boleh menulis: skill adalah alat kerja
-- bersama, bukan catatan pribadi. Kolom `created_by` merekam siapa yang
-- menambah, bukan untuk membatasi siapa yang bisa memakainya.

create table if not exists public.skills (
  id uuid primary key default gen_random_uuid(),
  -- Nama dipanggil dengan garis miring di chat: /cek-keamanan
  -- Huruf kecil, angka, dan tanda hubung saja supaya tidak ada spasi yang
  -- membingungkan saat dipanggil dari teks.
  nama text not null unique check (nama ~ '^[a-z0-9]([a-z0-9-]{0,62}[a-z0-9])?$'),
  ringkasan text not null default '' check (char_length(ringkasan) <= 200),
  isi text not null check (char_length(isi) between 1 and 20000),
  created_by uuid references auth.users (id) on delete set null,
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now()
);

-- Daftar skill selalu diurutkan nama; indeks unik sudah menutup ini, tetapi
-- indeks eksplisit membuat niatnya terbaca.
create index if not exists skills_nama_idx on public.skills (nama);

alter table public.skills enable row level security;

-- Semua pengguna terautentikasi boleh MEMBACA (skill alat kerja bersama).
drop policy if exists skills_baca on public.skills;
create policy skills_baca on public.skills
  for select to authenticated using (true);

-- Tulis hanya admin. Aplikasi Go memakai service role dan menegakkan aturan ini
-- juga di usecase, tetapi RLS menjaga jalur langsung ke Postgres tetap tertutup.
drop policy if exists skills_tulis_admin on public.skills;
create policy skills_tulis_admin on public.skills
  for all to authenticated
  using (exists (
    select 1 from public.profiles p
    where p.id = auth.uid() and p.is_admin
  ))
  with check (exists (
    select 1 from public.profiles p
    where p.id = auth.uid() and p.is_admin
  ));
