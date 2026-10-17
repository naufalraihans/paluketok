-- ============================================================
-- 0005: slug proyek untuk URL yang enak dibaca
-- ============================================================
-- Alasan: UUID di URL sulit dibaca dan dibagikan. Slug diturunkan dari nama
-- proyek dan unik PER OWNER (dua user boleh punya "kasir-warung" masing-masing).
--
-- Aturan:
--   * Kolom boleh kosong (nullable) supaya migrasi tidak memecah baris lama.
--   * Slug BEKU sejak dibuat: mengganti nama proyek tidak mengubah URL, jadi
--     tautan yang sudah dibagikan tidak pernah mati dan tidak perlu riwayat.
--   * Unik per (owner_id, slug) lewat index parsial: baris ber-slug NULL tidak
--     ikut terindeks, jadi aman walau ada baris lama yang belum punya slug.
--   * UUID tetap primary key dan tetap bisa dibuka di URL (halaman menukarnya
--     ke slug saat dimuat).

alter table public.projects
  add column if not exists slug text;

comment on column public.projects.slug is
  'Bagian URL proyek yang enak dibaca; unik per owner, beku sejak dibuat.';

-- Backfill proyek lama: slug diturunkan dari nama, dinomori bila bentrok dalam
-- owner yang sama. Aturan turunan SAMA dengan entity.Slugify: huruf kecil,
-- rentetan non-alfanumerik jadi satu '-', tanda hubung tepi dibuang.
with norm as (
  select id,
         owner_id,
         left(trim(both '-' from regexp_replace(lower(name), '[^a-z0-9]+', '-', 'g')), 60) as base
  from public.projects
  where slug is null
),
siap as (
  select id, owner_id, coalesce(nullif(base, ''), 'proyek') as base from norm
),
bernomor as (
  select id, owner_id, base,
         row_number() over (partition by owner_id, base order by id) as rn
  from siap
)
update public.projects p
set slug = case
             when r.rn = 1 then r.base
             else left(r.base, 55) || '-' || r.rn
           end
from bernomor r
where p.id = r.id;

create unique index if not exists uq_projects_owner_slug
  on public.projects (owner_id, slug)
  where slug is not null;
