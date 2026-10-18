package usecase

import (
	"context"

	"paluketok/server/internal/entity"
	"paluketok/server/internal/port"
)

// StageProposer menyusun usulan skema tahap dari brief + stack.
//
// Bukan port penuh: ini kebutuhan satu titik pemakaian (layar konfirmasi sebelum
// run dibuat), jadi kontraknya kecil dan hidup di sini, bukan di port.go.
type StageProposer interface {
	ProposeStages(ctx context.Context, brief string, stack map[string]string) ([]entity.PlanStage, string)
}

// StageStore menyimpan dan membaca skema tahap satu run.
type StageStore interface {
	ReplaceByRun(ctx context.Context, runID entity.ID, stages []entity.PlanStage) error
	ListByRun(ctx context.Context, runID entity.ID) ([]entity.PlanStage, error)
}

// SetStageProposer memasang penyusun skema tahap (implementasi: worker.Pipeline).
func (u *RunUsecase) SetStageProposer(p StageProposer, store StageStore) {
	u.proposer = p
	u.stages = store
}

// PreviewStages mengembalikan usulan skema tahap TANPA menyimpan dan TANPA
// memotong kredit: ini isi layar konfirmasi sebelum user menekan Mulai.
//
// ponytail: satu panggilan judge per preview. Kalau user menekan muat-ulang
// berkali-kali biayanya ikut berkali-kali; tambah cache per (project, brief)
// bila itu jadi keluhan nyata.
func (u *RunUsecase) PreviewStages(ctx context.Context, actor entity.Actor, projectID entity.ID) ([]entity.PlanStage, string, error) {
	if actor.UserID == "" {
		return nil, "", ErrForbidden
	}
	p, err := u.deps.Projects.GetByID(ctx, projectID)
	if err != nil {
		if err == port.ErrNotFound {
			return nil, "", ErrNotFound
		}
		return nil, "", wrap(err)
	}
	if p.OwnerID != actor.UserID && !actor.IsAdmin {
		return nil, "", ErrForbidden
	}
	stages, note := u.proposeFor(ctx, p, nil)
	return stages, note, nil
}

// PlanStagesOfRun membaca skema tahap satu run.
// Run lama tanpa baris skema mengembalikan susunan awal supaya UI tetap punya
// daftar tahap yang bisa ditampilkan.
func (u *RunUsecase) PlanStagesOfRun(ctx context.Context, actor entity.Actor, runID entity.ID) ([]entity.PlanStage, error) {
	if _, err := u.GetRun(ctx, actor, runID); err != nil {
		return nil, err
	}
	return u.stagesOfRun(ctx, runID), nil
}

// stagesOfRun pembacaan tanpa cek aktor, dipakai jalur internal.
func (u *RunUsecase) stagesOfRun(ctx context.Context, runID entity.ID) []entity.PlanStage {
	if u.stages == nil {
		return entity.DefaultPlanStages(runID)
	}
	list, err := u.stages.ListByRun(ctx, runID)
	if err != nil || len(list) == 0 {
		return entity.DefaultPlanStages(runID)
	}
	return list
}

// saveStages menulis skema ke run yang baru dibuat. Kegagalan TIDAK menggagalkan
// run: worker akan jatuh ke susunan awal dan run tetap jalan.
func (u *RunUsecase) saveStages(ctx context.Context, runID entity.ID, stages []entity.PlanStage) {
	if u.stages == nil || len(stages) == 0 {
		return
	}
	if err := u.stages.ReplaceByRun(ctx, runID, stages); err != nil {
		u.logger.Warn("simpan skema tahap gagal", "run", runID, "err", err)
	}
}

// proposeFor menentukan skema yang akan dipakai sebuah run.
//
// confirmed terisi bila user sudah menyunting/menyetujui lewat layar konfirmasi:
// saat itu usulan itu yang dipakai. Tanpa confirmed, judge ditanya lagi.
func (u *RunUsecase) proposeFor(ctx context.Context, p *entity.Project, confirmed []entity.PlanStage) ([]entity.PlanStage, string) {
	if len(confirmed) > 0 {
		clean, err := entity.NormalizePlanStages(confirmed)
		if err != nil {
			// Usulan yang dikonfirmasi ternyata tidak sah: jangan gagalkan run.
			return entity.DefaultPlanStages(""), "usulan dikonfirmasi ditolak, pakai susunan awal"
		}
		return clean, ""
	}
	if u.proposer == nil {
		return entity.DefaultPlanStages(""), ""
	}
	return u.proposer.ProposeStages(ctx, p.Brief, stackMap(p))
}

func stackMap(p *entity.Project) map[string]string {
	return map[string]string{"fe": p.Stack.FE, "be": p.Stack.BE, "db": p.Stack.DB}
}
