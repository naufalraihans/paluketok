// Kontrak handler -> worker. Dipisah agar delivery tidak mengimpor worker
// (arah dependensi: worker -> usecase/port; delivery -> port).
package httpapi

import "paluketok/server/internal/entity"

// Submitter menyodok run ke antrean worker. Penuh = error (dipetakan 503).
type Submitter interface {
	Submit(runID entity.ID) error
}
