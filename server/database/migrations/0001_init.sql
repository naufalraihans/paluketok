-- 0001_init.sql - PaluKetok
-- Sumber: plan/erd.dbml (sumber kebenaran skema), plan/erd.md (aturan di luar DBML),
-- plan/security.md bagian 3 (RLS + grant), plan/auth_billing.md 1.2 (trigger profiles).
-- Dijalankan sebagai postgres (owner). Supabase sudah menyediakan schema public, role anon/authenticated/service_role, dan auth.users.
-- Uji: workflow.md fase 2, langkah 8-10.

begin;

create extension if not exists pgcrypto;

-- ============================================================
-- profiles
-- ============================================================
create table if not exists public.profiles (
  id             uuid primary key,
  username       text unique,
  full_name      text,
  avatar_url     text,
  plan_tier      text not null default 'free',
  is_admin       boolean not null default false,
  credits        int  not null default 0 check (credits >= 0),
  token_version  int  not null default 0,
  created_at     timestamptz not null default now()
);
create index if not exists idx_profiles_admin on public.profiles (is_admin) where is_admin = true;

-- ============================================================
-- projects
-- ============================================================
create table if not exists public.projects (
  id          uuid primary key default gen_random_uuid(),
  owner_id    uuid not null references public.profiles(id) on delete cascade,
  name        text not null,
  brief       text not null check (char_length(brief) between 1 and 8000),
  stack       jsonb,
  status      text not null default 'draft' check (status in ('draft','ready','archived')),
  created_at  timestamptz not null default now(),
  updated_at  timestamptz not null default now()
);
create index if not exists idx_projects_owner_created on public.projects (owner_id, created_at);

-- ============================================================
-- plan_runs
-- ============================================================
create table if not exists public.plan_runs (
  id              uuid primary key default gen_random_uuid(),
  project_id      uuid not null references public.projects(id) on delete cascade,
  user_id         uuid not null references public.profiles(id) on delete cascade,
  status          text not null default 'queued' check (status in ('queued','running','done','failed')),
  current_stage   text check (current_stage in ('prd','erd','openapi','roles','security','structure')),
  progress        int not null default 0 check (progress between 0 and total),
  total           int not null default 6,
  escalations     int not null default 0,
  credits_charged int not null default 1,
  model           text,
  tokens_in       int not null default 0,
  tokens_out      int not null default 0,
  cost            numeric not null default 0,
  error           text,
  started_at      timestamptz,
  finished_at     timestamptz,
  created_at      timestamptz not null default now()
);
-- Satu run running per project (jaring kedua; penegak utama di usecase).
create unique index if not exists uq_plan_runs_one_running
  on public.plan_runs (project_id) where status = 'running';
create index if not exists idx_plan_runs_user_created on public.plan_runs (user_id, created_at);

-- ============================================================
-- artifacts
-- ============================================================
create table if not exists public.artifacts (
  id          uuid primary key default gen_random_uuid(),
  run_id      uuid not null references public.plan_runs(id) on delete cascade,
  user_id     uuid not null references public.profiles(id) on delete cascade,
  kind        text not null check (kind in ('prd','erd','openapi','roles','security','structure')),
  content     text not null,
  version     int not null default 1,
  status      text not null default 'draft' check (status in ('draft','final','stale')),
  created_at  timestamptz not null default now(),
  updated_at  timestamptz not null default now()
);
create unique index if not exists uq_artifacts_run_kind on public.artifacts (run_id, kind);
create index if not exists idx_artifacts_user_created on public.artifacts (user_id, created_at);

-- ============================================================
-- artifact_versions
-- ============================================================
create table if not exists public.artifact_versions (
  id                 uuid primary key default gen_random_uuid(),
  artifact_id        uuid not null references public.artifacts(id) on delete cascade,
  version            int not null,
  content            text not null,
  created_by_run_id  uuid references public.plan_runs(id) on delete set null,
  created_at         timestamptz not null default now(),
  unique (artifact_id, version)
);
-- Catatan: unique (artifact_id, version) sama saja dengan uq_artifact_versions di DBML.

-- ============================================================
-- decisions
-- ============================================================
create table if not exists public.decisions (
  id               uuid primary key default gen_random_uuid(),
  run_id           uuid not null references public.plan_runs(id) on delete cascade,
  user_id          uuid not null references public.profiles(id) on delete cascade,
  stage            text not null,
  question         text not null,
  state            jsonb,
  criteria         jsonb,
  gate             text not null check (gate in ('EXECUTE','GATHER_EVIDENCE','ESCALATE')),
  round            int not null default 1,
  choice           text,
  confidence       numeric check (confidence is null or (confidence >= 0 and confidence <= 1)),
  probabilities    jsonb,
  jev_decision_id  text,
  created_at       timestamptz not null default now()
);
create index if not exists idx_decisions_run_stage on public.decisions (run_id, stage);
create index if not exists idx_decisions_user_created on public.decisions (user_id, created_at);

-- ============================================================
-- orders (DOKU)
-- ============================================================
create table if not exists public.orders (
  id              uuid primary key default gen_random_uuid(),
  user_id         uuid not null references public.profiles(id) on delete restrict,
  package_id      text not null,
  invoice_number  text not null unique check (invoice_number ~ '^[A-Za-z0-9-]{1,30}$'),
  amount_idr      int not null check (amount_idr > 0),
  credits_granted int not null check (credits_granted > 0),
  status          text not null default 'pending' check (status in ('pending','settlement','expire')),
  request_id      text not null unique,
  payment_url     text,
  expired_date    timestamptz,
  paid_at         timestamptz,
  created_at      timestamptz not null default now(),
  updated_at      timestamptz not null default now()
);
create index if not exists idx_orders_user_created on public.orders (user_id, created_at);
create index if not exists idx_orders_status on public.orders (status);

-- ============================================================
-- credits_ledger
-- ============================================================
create table if not exists public.credits_ledger (
  id               uuid primary key default gen_random_uuid(),
  user_id          uuid not null references public.profiles(id) on delete restrict,
  delta            int not null check (delta <> 0),
  reason           text not null check (reason in ('topup','run','refund','regenerate','grant')),
  ref_type         text not null default 'none' check (ref_type in ('order','plan_run','artifact','ledger','none')),
  ref_id           uuid,
  actor_id         uuid references public.profiles(id) on delete set null,
  idempotency_key  text,
  created_at       timestamptz not null default now()
);
create index if not exists idx_ledger_user_created on public.credits_ledger (user_id, created_at);
create index if not exists idx_ledger_ref on public.credits_ledger (ref_type, ref_id);

-- Index penjaga uang (erd.dbml 150-153). Predikat ref_id is not null wajib:
-- Postgres menganggap NULL distinct, jadi tanpa predikat ini jaringnya bocor.
create unique index if not exists uq_ledger_one_refund_per_run
  on public.credits_ledger (ref_id)
  where reason = 'refund' and ref_type = 'plan_run' and ref_id is not null;
create unique index if not exists uq_ledger_one_topup_per_order
  on public.credits_ledger (ref_id)
  where reason = 'topup' and ref_type = 'order' and ref_id is not null;
create unique index if not exists uq_ledger_one_refund_per_charge
  on public.credits_ledger (ref_id)
  where reason = 'refund' and ref_type = 'ledger' and ref_id is not null;
create unique index if not exists uq_ledger_one_grant_per_key
  on public.credits_ledger (idempotency_key)
  where reason = 'grant' and idempotency_key is not null;

-- ============================================================
-- ai_settings
-- ============================================================
create table if not exists public.ai_settings (
  id                 uuid primary key default gen_random_uuid(),
  slot               text not null unique,
  provider           text not null,
  base_url           text not null,
  model              text not null,
  api_key            text not null,
  max_input_tokens   int not null check (max_input_tokens > 0),
  max_output_tokens  int not null check (max_output_tokens > 0),
  is_active          boolean not null default true,
  updated_by         uuid references public.profiles(id) on delete set null,
  updated_at         timestamptz not null default now()
);

-- ============================================================
-- default_tier dummy check (tanpa-artefak trap): hanya cek east
-- ============================================================
-- updated_at otomatis
create or replace function public.touch_updated_at()
returns trigger language plpgsql as $$
begin
  new.updated_at := now();
  return new;
end $$;

drop trigger if exists trg_projects_updated on public.projects;
create trigger trg_projects_updated before update on public.projects
  for each row execute function public.touch_updated_at();

drop trigger if exists trg_artifacts_updated on public.artifacts;
create trigger trg_artifacts_updated before update on public.artifacts
  for each row execute function public.touch_updated_at();

drop trigger if exists trg_orders_updated on public.orders;
create trigger trg_orders_updated before update on public.orders
  for each row execute function public.touch_updated_at();

-- ============================================================
-- trigger profiles: auth.users insert -> profiles insert
-- plan/auth_billing.md 1.2. SECURITY DEFINER + search_path terkunci.
-- ============================================================
create or replace function public.handle_new_user()
returns trigger
language plpgsql
security definer
set search_path = ''
as $$
declare
  base_username text;
  new_username  text;
  attempt       int := 1;
begin
  base_username := coalesce(
    nullif(new.raw_user_meta_data ->> 'username', ''),
    split_part(new.email, '@', 1)
  );
  -- sanity: buang karakter aneh, batasi panjang
  base_username := regexp_replace(base_username, '[^A-Za-z0-9_.-]', '', 'g');
  if base_username = '' then
    base_username := 'user';
  end if;
  base_username := left(base_username, 40);

  new_username := base_username;
  while exists (select 1 from public.profiles where username = new_username) loop
    attempt := attempt + 1;
    new_username := base_username || attempt;
    if attempt > 100 then
      -- ponytail: collision 100 kali = hampir mustahil; fallback uuid pendek
      new_username := left(base_username, 30) || '-' || left(new.id::text, 8);
      exit;
    end if;
  end loop;

  insert into public.profiles (id, username, full_name, avatar_url, plan_tier, is_admin, credits, token_version)
  values (new.id, new_username, coalesce(nullif(new.raw_user_meta_data ->> 'full_name',''), ''), nullif(new.raw_user_meta_data ->> 'avatar_url',''), 'free', false, 0, 0);
  return new;
end $$;

drop trigger if exists on_auth_user_created on auth.users;
create trigger on_auth_user_created
  after insert on auth.users
  for each row execute function public.handle_new_user();

-- ============================================================
-- RLS: enable + revoke + policy (security.md bagian 3)
-- ============================================================
-- 1) Enable RLS di semua tabel app
alter table public.profiles          enable row level security;
alter table public.projects          enable row level security;
alter table public.plan_runs         enable row level security;
alter table public.artifacts         enable row level security;
alter table public.artifact_versions enable row level security;
alter table public.decisions         enable row level security;
alter table public.orders            enable row level security;
alter table public.credits_ledger    enable row level security;
alter table public.ai_settings       enable row level security;

-- 2) Cabut semua grant default dari anon/authenticated (PostgREST access)
revoke all on all tables in schema public from anon;
revoke all on all tables in schema public from authenticated;

-- 3) Policy
-- profiles: baca baris sendiri; update kolom terbatas (username, full_name, avatar_url)
-- kolom is_admin, credits, plan_tier, token_version dilindungi lewat trigger kolom
-- (grant UPDATE di-cabut lalu dikembalikan penuh hanya supaya owner bisa update beberapa kolom).
drop policy if exists "profiles_select_own" on public.profiles;
create policy "profiles_select_own" on public.profiles
  for select using (id = auth.uid());

drop policy if exists "profiles_update_own" on public.profiles;
create policy "profiles_update_own" on public.profiles
  for update using (id = auth.uid()) with check (id = auth.uid());

-- Guard kolom sensitif: tolak UPDATE yang mengubah kolom selain username/full_name/avatar_url
create or replace function public.guard_profiles_update()
returns trigger language plpgsql as $$
begin
  if new.is_admin      is distinct from old.is_admin
  or new.credits       is distinct from old.credits
  or new.plan_tier     is distinct from old.plan_tier
  or new.token_version is distinct from old.token_version
  or new.id            is distinct from old.id
  or new.created_at    is distinct from old.created_at then
    raise exception 'profiles: kolom ini hanya bisa diubah lewat service role';
  end if;
  return new;
end $$;
drop trigger if exists trg_guard_profiles_update on public.profiles;
create trigger trg_guard_profiles_update before update on public.profiles
  for each row execute function public.guard_profiles_update();

-- projects: owner (owner_id)
drop policy if exists "projects_all_own" on public.projects;
create policy "projects_all_own" on public.projects
  for all using (owner_id = auth.uid()) with check (owner_id = auth.uid());

-- plan_runs, artifacts, decisions: user_id
drop policy if exists "plan_runs_all_own" on public.plan_runs;
create policy "plan_runs_all_own" on public.plan_runs
  for all using (user_id = auth.uid()) with check (user_id = auth.uid());
drop policy if exists "artifacts_all_own" on public.artifacts;
create policy "artifacts_all_own" on public.artifacts
  for all using (user_id = auth.uid()) with check (user_id = auth.uid());
drop policy if exists "decisions_all_own" on public.decisions;
create policy "decisions_all_own" on public.decisions
  for all using (user_id = auth.uid()) with check (user_id = auth.uid());

-- artifact_versions: policy lewat join artifacts
drop policy if exists "artifact_versions_all_own" on public.artifact_versions;
create policy "artifact_versions_all_own" on public.artifact_versions
  for all using (
    artifact_id in (select id from public.artifacts where user_id = auth.uid())
  ) with check (
    artifact_id in (select id from public.artifacts where user_id = auth.uid())
  );

-- orders: select klien boleh lihat order sendiri (poll status); insert tidak lewat DB (lewat Go api)
drop policy if exists "orders_select_own" on public.orders;
create policy "orders_select_own" on public.orders
  for select using (user_id = auth.uid());

-- credits_ledger: nol akses dari anon/authenticated (lewat revoke di atas)
-- ai_settings: nol akses dari anon/authenticated (lewat revoke di atas)

commit;

-- ============================================================
-- Verifikasi manual (dijalankan terpisah, bukan dalam transaksi):
--   \d profiles  \d projects  \d plan_runs  \d artifacts
--   \d artifact_versions  \d decisions  \d orders  \d credits_ledger  \d ai_settings
-- ============================================================
