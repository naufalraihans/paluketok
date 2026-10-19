-- 0008: sesi chat per proyek.
--
-- Sebelumnya semua pesan satu proyek dianggap satu percakapan, jadi riwayat yang
-- dikirim ke model selalu seluruh proyek. Itu dua masalah: konteks membengkak,
-- dan topik lama ikut terbawa ke pertanyaan baru.
--
-- Sekarang tiap percakapan punya baris sendiri. Pesan lama (tanpa sesi) tetap
-- dipertahankan: kolomnya nullable, dan aplikasi membuatkan satu sesi
-- "Percakapan awal" per proyek yang punya pesan lama, lalu mengisinya.
--
-- Catatan RLS: `create table` TIDAK mengaktifkan row level security. Harus
-- eksplisit `enable row level security` + `revoke`, pola sama dengan 0006/0007.
-- Tanpa itu tabel baru bisa dibaca anon lewat PostgREST.

create table if not exists public.chat_sessions (
  id          uuid primary key default gen_random_uuid(),
  project_id  uuid not null references public.projects(id) on delete cascade,
  user_id     uuid not null references public.profiles(id) on delete cascade,
  judul       text not null check (char_length(judul) between 1 and 120),
  created_at  timestamptz not null default now(),
  updated_at  timestamptz not null default now()
);

comment on table public.chat_sessions is
  'Sesi percakapan perancangan per proyek. Satu proyek boleh punya banyak sesi.';

-- Daftar sesi dibaca per proyek, terbaru dulu.
create index if not exists idx_chat_sessions_project
  on public.chat_sessions (project_id, updated_at desc);

-- Pesan menempel ke sesinya. Nullable supaya pesan lama tidak hilang saat
-- migrasi; aplikasi mengisinya lewat backfill di bawah.
alter table public.project_messages
  add column if not exists session_id uuid references public.chat_sessions(id) on delete cascade;

-- Satu proyek dengan pesan lama = satu sesi awal. Idempoten: hanya proyek yang
-- belum punya sesi sama sekali yang dibuatkan.
--
-- min(m.user_id) TIDAK bisa dipakai: Postgres tidak punya agregat min() untuk
-- uuid. Diambil satu baris pemilik pesan secara deterministik lewat distinct on.
insert into public.chat_sessions (project_id, user_id, judul, created_at, updated_at)
select f.project_id, f.user_id, 'Percakapan awal', f.first_at, f.last_at
from (
  select distinct on (m.project_id)
         m.project_id, m.user_id, m.created_at as first_at,
         max(m.created_at) over (partition by m.project_id) as last_at
  from public.project_messages m
  where not exists (
    select 1 from public.chat_sessions s where s.project_id = m.project_id
  )
  order by m.project_id, m.created_at
) f;

-- Pesan tanpa sesi dimasukkan ke sesi proyeknya.
update public.project_messages m
set session_id = s.id
from public.chat_sessions s
where m.session_id is null and s.project_id = m.project_id;

-- Riwayat satu sesi dibaca urut waktu.
create index if not exists idx_project_messages_session
  on public.project_messages (session_id, created_at);

alter table public.chat_sessions enable row level security;
revoke all on public.chat_sessions from anon, authenticated;

drop policy if exists "chat_sessions_all_own" on public.chat_sessions;
create policy "chat_sessions_all_own" on public.chat_sessions
  for all using (user_id = auth.uid()) with check (user_id = auth.uid());
