// Daftar artefak per project untuk sidebar progresif.
//
// Kenapa perlu: GET /v1/runs/{id}/artifacts butuh run id, dan frontend
// menyimpannya di sessionStorage. Akibatnya proyek lama dibuka di browser baru
// = sidebar kosong walau artefaknya ada di server. Endpoint ini membaca artefak
// lewat project, jadi tidak bergantung pada run id yang dikirim klien.
package usecase

import (
	"context"
	"errors"

	"paluketok/server/internal/entity"
	"paluketok/server/internal/port"
)

// ListProjectArtifacts mengembalikan artefak terbaru per kind untuk satu
// project milik actor. ref boleh UUID atau slug. Policy (owner atau admin)
// ditegakkan lewat resolveProject, sama seperti GET /v1/projects/{id}.
func (u *ProjectUsecase) ListProjectArtifacts(ctx context.Context, actor entity.Actor, ref string) ([]entity.Artifact, error) {
	p, err := u.resolveProject(ctx, actor, ref)
	if err != nil {
		return nil, err
	}
	list, err := u.deps.Artifacts.ListLatestByProject(ctx, p.ID)
	if err != nil {
		if errors.Is(err, port.ErrNotFound) {
			return nil, ErrNotFound
		}
		return nil, wrap(err)
	}
	return list, nil
}

// ResolveProjectID menerjemahkan ref URL (UUID atau slug) menjadi UUID proyek,
// sambil menegakkan policy owner/admin. Dipakai rute yang butuh UUID asli
// (POST /v1/projects/{ref}/runs, PATCH, DELETE) supaya klien boleh memakai
// bentuk slug dan tidak wajib tahu UUID.
func (u *ProjectUsecase) ResolveProjectID(ctx context.Context, actor entity.Actor, ref string) (entity.ID, error) {
	p, err := u.resolveProject(ctx, actor, ref)
	if err != nil {
		return "", err
	}
	return p.ID, nil
}
