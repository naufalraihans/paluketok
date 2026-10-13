package repository

// Integration test ke Supabase sandbox (DATABASE_URL di server/.env).
// Aturan: data uji dibuat sendiri, dibersihkan sendiri, tidak ada sisa.
// Jalur singkat: go test ./internal/repository/ -count=1
//
// Keunikan data uji: UUID v4 deterministik per run test dengan prefix tetap
// a4f4xxxx supaya cleanup bisa menghapus milik test ini saja.

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"paluketok/server/internal/entity"
	"paluketok/server/internal/port"
)

const testUserA = "a4f40000-0000-4000-8000-00000000a100"

func testDB(t *testing.T) (*pgxpool.Pool, *Repos) {
	t.Helper()
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL kosong: integration test dilewati")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("buka pool: %v", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		t.Fatalf("ping sandbox: %v", err)
	}
	t.Cleanup(func() { cleanupTestRows(context.Background(), t, pool); pool.Close() })
	return pool, New(pool)
}

// cleanupTestRows hapus semua baris uji: user a4f4... cascade ke tabel anak
// (erd.dbml delete cascade), kecuali orders/ledger (restrict) dihapus manual
// lebih dulu. Kalau test sebelum user dibuat gagal, tidak ada yang perlu dihapus.
func cleanupTestRows(ctx context.Context, t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	for _, q := range []string{
		`delete from credits_ledger where user_id='` + testUserA + `' or actor_id='` + testUserA + `'`,
		`delete from orders where user_id='` + testUserA + `'`,
		`delete from artifacts where user_id='` + testUserA + `'`,
		`delete from plan_runs where user_id='` + testUserA + `'`,
		`delete from projects where owner_id='` + testUserA + `'`,
		`delete from profiles where id='` + testUserA + `'`,
		`delete from auth.users where id='` + testUserA + `'`,
	} {
		if _, err := pool.Exec(ctx, q); err != nil {
			t.Errorf("cleanup: %v (query: %s)", err, q)
		}
	}
}

// mustProfile membuat user uji di auth.users (supaya FK profiles terpenuhi,
// tanpa mengirim email) lalu baris profiles dengan saldo tertentu. Trigger
// profiles_sinkron buat profiles; kita update credits setelahnya.
func mustProfile(t *testing.T, pool *pgxpool.Pool, credits int) {
	t.Helper()
	_, err := pool.Exec(context.Background(),
		`insert into auth.users (instance_id, id, aud, role, email, encrypted_password,
		   email_confirmed_at, created_at, updated_at,
		   raw_app_meta_data, raw_user_meta_data)
		 values ('00000000-0000-0000-0000-000000000000', $1, 'authenticated', 'authenticated',
		   $2, 'x', now(), now(), now(), '{"provider":"email","providers":["email"]}', '{}')
		 on conflict (id) do nothing`,
		testUserA, "uji-f4@example.com")
	if err != nil {
		t.Fatalf("seed auth.users: %v", err)
	}
	// trigger menyalin baris profiles; pastikan ada (idempoten rerun)
	if _, err := pool.Exec(context.Background(),
		`insert into profiles (id, username) values ($1, $2)
		 on conflict (id) do nothing`,
		testUserA, "uji-f4"); err != nil {
		t.Fatalf("seed profiles: %v", err)
	}
	if _, err := pool.Exec(context.Background(),
		`update profiles set credits=$2 where id=$1`, testUserA, credits); err != nil {
		t.Fatalf("seed credits: %v", err)
	}
}

func mustProject(t *testing.T, repos *Repos, now time.Time) *entity.Project {
	t.Helper()
	id := entity.ID("a4f40000-0000-4000-8000-00000000c001") // harus UUID valid
	p := &entity.Project{
		ID: id, OwnerID: testUserA, Name: "uji-f4", Brief: "project uji integration",
		Stack:  entity.Stack{FE: "svelte", BE: "go", DB: "supabase", Payment: "doku"},
		Status: entity.ProjectDraft, CreatedAt: now, UpdatedAt: now,
	}
	if err := repos.Projects.Create(context.Background(), p); err != nil {
		t.Fatalf("buat project: %v", err)
	}
	return p
}

func newTestRun(p *entity.Project, now time.Time) *entity.PlanRun {
	return &entity.PlanRun{
		ID:             entity.ID(newUUID()),
		ProjectID:      p.ID,
		UserID:         testUserA,
		Status:         entity.RunQueued,
		Total:          entity.TotalStages,
		CreditsCharged: 1,
		CreatedAt:      now,
	}
}

// ===== potong kredit (charge) =====

func TestIntegration_ChargeForRun_PotongDanCatat(t *testing.T) {
	pool, repos := testDB(t)
	mustProfile(t, pool, 5)
	p := mustProject(t, repos, time.Now())
	run := newTestRun(p, time.Now())

	if err := repos.Ledger.ChargeForRun(context.Background(), testUserA, run); err != nil {
		t.Fatalf("charge: %v", err)
	}
	var credits int
	if err := pool.QueryRow(context.Background(),
		`select credits from profiles where id=$1`, testUserA).Scan(&credits); err != nil || credits != 4 {
		t.Fatalf("saldo harus 4, dapat %d (err=%v)", credits, err)
	}
	// ledger: satu baris -1 reason run
	var n int
	if err := pool.QueryRow(context.Background(),
		`select count(*) from credits_ledger where user_id=$1 and reason='run' and delta=-1 and ref_id=$2`,
		testUserA, run.ID).Scan(&n); err != nil || n != 1 {
		t.Fatalf("ledger run harus 1 baris, dapat %d (err=%v)", n, err)
	}
}

func TestIntegration_ChargeForRun_SaldoKurang402(t *testing.T) {
	pool, repos := testDB(t)
	mustProfile(t, pool, 0)
	p := mustProject(t, repos, time.Now())
	run := newTestRun(p, time.Now())

	err := repos.Ledger.ChargeForRun(context.Background(), testUserA, run)
	if err != port.ErrInsufficientCredits {
		t.Fatalf("harus 402, dapat %v", err)
	}
	var n int
	pool.QueryRow(context.Background(),
		`select count(*) from plan_runs where id=$1`, run.ID).Scan(&n)
	if n != 0 {
		t.Fatalf("run tidak boleh tercipta saat saldo kurang")
	}
}

// Dua charge serentak tidak boleh melewati saldo (FOR UPDATE, security.md 7).
func TestIntegration_ChargeForRun_KonkurensiTidakNegatif(t *testing.T) {
	pool, repos := testDB(t)
	mustProfile(t, pool, 5)
	p := mustProject(t, repos, time.Now())

	const paralel = 8
	errCh := make(chan error, paralel)
	for i := 0; i < paralel; i++ {
		go func() {
			run := newTestRun(p, time.Now())
			errCh <- repos.Ledger.ChargeForRun(context.Background(), testUserA, run)
		}()
	}
	okCount := 0
	for i := 0; i < paralel; i++ {
		err := <-errCh
		if err == nil {
			okCount++
			continue
		}
		if err != port.ErrInsufficientCredits && err != port.ErrConflict {
			// conflict bila dua run menabrak uq_plan_runs_one_running? tidak,
			// semua queued; satu project boleh banyak queued. Jadi selain dua
			// error di atas itu bug.
			t.Fatalf("error tak terduga saat paralel: %v", err)
		}
	}
	var credits int
	pool.QueryRow(context.Background(), `select credits from profiles where id=$1`, testUserA).Scan(&credits)
	if credits != 5-okCount {
		t.Fatalf("rekonstruksi: saldo %d, sukses %d (harus 5-%d)", credits, okCount, okCount)
	}
	if credits < 0 {
		t.Fatalf("saldo negatif! %d", credits)
	}
}

// ===== refund =====

func TestIntegration_RefundRun_SekaliSaja(t *testing.T) {
	pool, repos := testDB(t)
	mustProfile(t, pool, 5)
	p := mustProject(t, repos, time.Now())
	run := newTestRun(p, time.Now())
	if err := repos.Ledger.ChargeForRun(context.Background(), testUserA, run); err != nil {
		t.Fatalf("charge: %v", err)
	}
	run.Status = entity.RunFailed
	if err := repos.Runs.Update(context.Background(), run); err != nil {
		t.Fatal(err)
	}

	if err := repos.Ledger.RefundRun(context.Background(), run, 1); err != nil {
		t.Fatalf("refund: %v", err)
	}
	// refund kedua = no-op
	if err := repos.Ledger.RefundRun(context.Background(), run, 1); err != nil {
		t.Fatalf("refund kedua harus no-op: %v", err)
	}
	var credits, nRefund int
	pool.QueryRow(context.Background(), `select credits from profiles where id=$1`, testUserA).Scan(&credits)
	pool.QueryRow(context.Background(),
		`select count(*) from credits_ledger where ref_id=$1 and reason='refund'`, run.ID).Scan(&nRefund)
	if credits != 5 || nRefund != 1 {
		t.Fatalf("saldo harus 5 (2-1+1 refund sekali... charge 5->4 refund 4->5), dapat %d, baris refund %d", credits, nRefund)
	}
}

// ===== grant admin idempoten =====

func TestIntegration_Grant_Idempoten(t *testing.T) {
	pool, repos := testDB(t)
	mustProfile(t, pool, 0)

	if err := repos.Ledger.Grant(context.Background(), testUserA, 7, testUserA, "uji-f4-grant-key-1"); err != nil {
		t.Fatalf("grant: %v", err)
	}
	if err := repos.Ledger.Grant(context.Background(), testUserA, 7, testUserA, "uji-f4-grant-key-1"); err != nil {
		t.Fatalf("replay harus no-op: %v", err)
	}
	var credits, n int
	pool.QueryRow(context.Background(), `select credits from profiles where id=$1`, testUserA).Scan(&credits)
	pool.QueryRow(context.Background(),
		`select count(*) from credits_ledger where reason='grant' and idempotency_key='uji-f4-grant-key-1'`).Scan(&n)
	if credits != 7 || n != 1 {
		t.Fatalf("replay grant menambah ganda: credits %d, baris %d", credits, n)
	}
}

// ===== settle topup: transisi WHERE status='pending' =====

func TestIntegration_SettleTopUp_SekaliDanGuardPending(t *testing.T) {
	pool, repos := testDB(t)
	mustProfile(t, pool, 0)
	now := time.Now()
	o := &entity.Order{
		ID: entity.ID("a4f40000-0000-4000-8000-00000000d001"), UserID: testUserA,
		PackageID: "starter", InvoiceNumber: "PK-uji-f4-d001", AmountIDR: 25000,
		CreditsGranted: 10, Status: entity.OrderPending,
		RequestID: "a4f40000-0000-4000-8000-00000000d001",
		CreatedAt: now, UpdatedAt: now,
	}
	if err := repos.Orders.Create(context.Background(), o); err != nil {
		t.Fatalf("buat order: %v", err)
	}

	if err := repos.Ledger.SettleTopUp(context.Background(), o); err != nil {
		t.Fatalf("settle: %v", err)
	}
	// settle kedua HARUS ditolak (bukan pending lagi)
	if err := repos.Ledger.SettleTopUp(context.Background(), o); err != port.ErrConflict {
		t.Fatalf("settle kedua harus conflict, dapat %v", err)
	}
	var credits, nTopup int
	var status string
	pool.QueryRow(context.Background(), `select credits from profiles where id=$1`, testUserA).Scan(&credits)
	pool.QueryRow(context.Background(), `select status from orders where id=$1`, o.ID).Scan(&status)
	pool.QueryRow(context.Background(),
		`select count(*) from credits_ledger where reason='topup' and ref_id=$1`, o.ID).Scan(&nTopup)
	if credits != 10 || nTopup != 1 || status != "settlement" {
		t.Fatalf("replay settle mengubah data: credits %d, topup %d, status %q", credits, nTopup, status)
	}
}

// expire hanya dari pending
func TestIntegration_ExpireOrder_DariPendingSaja(t *testing.T) {
	pool, repos := testDB(t)
	mustProfile(t, pool, 0)
	now := time.Now()

	// order pending -> boleh expire
	o1 := &entity.Order{
		ID: entity.ID("a4f40000-0000-4000-8000-00000000d002"), UserID: testUserA,
		PackageID: "starter", InvoiceNumber: "PK-uji-f4-d002", AmountIDR: 25000,
		CreditsGranted: 10, Status: entity.OrderPending, RequestID: "a4f40000-0000-4000-8000-00000000d002",
		CreatedAt: now, UpdatedAt: now,
	}
	if err := repos.Orders.Create(context.Background(), o1); err != nil {
		t.Fatal(err)
	}
	tag, err := pool.Exec(context.Background(),
		`update orders set status='expire', updated_at=now() where id=$1 and status='pending'`, o1.ID)
	if err != nil {
		t.Fatal(err)
	}
	if tag.RowsAffected() != 1 {
		t.Fatal("pending -> expire harus 1 baris")
	}

	// order settlement -> expire DITOLAK (RowsAffected 0)
	o2 := &entity.Order{
		ID: entity.ID("a4f40000-0000-4000-8000-00000000d003"), UserID: testUserA,
		PackageID: "starter", InvoiceNumber: "PK-uji-f4-d003", AmountIDR: 25000,
		CreditsGranted: 10, Status: entity.OrderSettlement, RequestID: "a4f40000-0000-4000-8000-00000000d003",
		CreatedAt: now, UpdatedAt: now,
	}
	if err := repos.Orders.Create(context.Background(), o2); err != nil {
		t.Fatal(err)
	}
	tag2, err := pool.Exec(context.Background(),
		`update orders set status='expire', updated_at=now() where id=$1 and status='pending'`, o2.ID)
	if err != nil {
		t.Fatal(err)
	}
	if tag2.RowsAffected() != 0 {
		t.Fatal("settlement tidak boleh turun ke expire")
	}
	st, _ := repos.Orders.GetByID(context.Background(), o2.ID)
	if st.Status != entity.OrderSettlement {
		t.Fatalf("status harus tetap settlement, dapat %q", st.Status)
	}
}

// ===== profil & project =====

func TestIntegration_ProfileDanProjectCRUD(t *testing.T) {
	pool, repos := testDB(t)
	mustProfile(t, pool, 3)

	got, err := repos.Profiles.GetByID(context.Background(), testUserA)
	if err != nil || got.Credits != 3 {
		t.Fatalf("get profile: %v credits %d", err, got.Credits)
	}

	now := time.Now()
	p := mustProject(t, repos, now)
	gotP, err := repos.Projects.GetByID(context.Background(), p.ID)
	if err != nil {
		t.Fatalf("get project: %v", err)
	}
	if gotP.Stack.FE != "svelte" || gotP.Stack.Payment != "doku" {
		t.Fatalf("stack jsonb salah: %+v", gotP.Stack)
	}
	list, err := repos.Projects.ListByOwner(context.Background(), testUserA, port.Page{Limit: 10})
	if err != nil || len(list) != 1 {
		t.Fatalf("list: %v len %d", err, len(list))
	}
}
