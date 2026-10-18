// Gate JEV (docs/plan/planner.md bagian 4): dipanggil hanya di titik ragu.
// >= 0.90 EXECUTE; 0.50-0.89 GATHER (tambah evidence, maks 4 putaran);
// < 0.50 atau putaran ke-4 masih < 0.90 = ESCALATE (run lanjut, default stack).
// Timeout 30s dan tanpa retry ada di adapter judge; timeout dianggap ESCALATE.
package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"paluketok/server/internal/entity"
	"paluketok/server/internal/port"
)

// judgeTimeout sesuai planner.md bagian 4 (gateway 30 detik). Ditegakkan di
// sini juga supaya worker tidak pernah menggantung pada JEV.
const judgeTimeout = 30 * time.Second

// GateQuestion satu titik ragu pada tahap tertentu.
type GateQuestion struct {
	Question string
	// Options pilihan wajar; Default dipakai saat ESCALATE.
	Options []string
	Default string
}

// GateResult hasil gate untuk satu pertanyaan.
type GateResult struct {
	Gate       entity.Gate
	Choice     string
	Confidence float64
}

// runGate menjalankan gate untuk satu pertanyaan dengan evidence awal.
// setiap panggilan JEV dicatat ke decisions (question, state, criteria, gate,
// choice, confidence, probabilities). Mengembalikan pilihan terpakai.
func (p *Pipeline) runGate(ctx context.Context, run *entity.PlanRun, stage entity.Stage, q GateQuestion) (GateResult, error) {
	evidence := map[string]any{
		"brief":     p.brief,
		"stage":     string(stage),
		"stack":     p.stackSummary,
		"artifacts": p.artifactDigests(),
		"options":   q.Options,
	}
	res := GateResult{Gate: entity.GateEscalate, Choice: q.Default}

	for round := 1; round <= entity.MaxGateRounds; round++ {
		cctx, cancel := context.WithTimeout(ctx, judgeTimeout)
		resp, err := p.judge.Decide(cctx, buildJudgeRequest(q, evidence))
		cancel()

		ans, has := judgeAnswer(resp, q)
		switch {
		case err != nil || !has:
			// Timeout/error gateway = ESCALATE tanpa retry (planner.md 4).
			p.recordDecision(ctx, run, stage, q, round, entity.GateEscalate, "", 0, resp, err)
			return res, nil
		case ans.Confidence >= 0.90:
			res = GateResult{Gate: entity.GateExecute, Choice: ans.Choice, Confidence: ans.Confidence}
			p.recordDecision(ctx, run, stage, q, round, entity.GateExecute, ans.Choice, ans.Confidence, resp, nil)
			return res, nil
		case ans.Confidence >= 0.50:
			added := p.addEvidence(evidence, q, ans, stage)
			if round == entity.MaxGateRounds || !added {
				// Putaran ke-4 masih < 0.90, atau tak ada fakta baru.
				res = GateResult{Gate: entity.GateEscalate, Choice: q.Default, Confidence: ans.Confidence}
				p.recordDecision(ctx, run, stage, q, round, entity.GateEscalate, q.Default, ans.Confidence, resp, nil)
				return res, nil
			}
			p.recordDecision(ctx, run, stage, q, round, entity.GateGatherEvidence, "", ans.Confidence, resp, nil)
		default: // < 0.50 ESCALATE, pakai default, run lanjut
			res = GateResult{Gate: entity.GateEscalate, Choice: q.Default, Confidence: ans.Confidence}
			p.recordDecision(ctx, run, stage, q, round, entity.GateEscalate, q.Default, ans.Confidence, resp, nil)
			return res, nil
		}
	}
	return res, nil
}

// buildJudgeRequest menyiapkan request JEV: state ringkas, satu pertanyaan.
func buildJudgeRequest(q GateQuestion, evidence map[string]any) port.JudgeRequest {
	criteria := map[string]any{"options": q.Options}
	return port.JudgeRequest{
		State: evidence,
		Questions: []port.JudgeQuestion{{
			ID:           "q1",
			Type:         "choice",
			Instructions: q.Question,
			Criteria:     criteria,
		}},
	}
}

// judgeAnswer mengekstrak jawaban q1 dari respons (nil aman).
func judgeAnswer(resp *port.JudgeResponse, _ GateQuestion) (port.JudgeAnswer, bool) {
	if resp == nil {
		return port.JudgeAnswer{}, false
	}
	a, ok := resp.Answers["q1"]
	return a, ok
}

// addEvidence menambah fakta baru. false bila tidak ada yang bisa ditambah
// (planner.md 4: "GATHER menambah fakta, bukan mengulang prompt yang sama").
// Sumber fakta: brief (kata kunci opsi), ringkasan artefak, dan stack.
func (p *Pipeline) addEvidence(evidence map[string]any, q GateQuestion, ans port.JudgeAnswer, stage entity.Stage) bool {
	key := fmt.Sprintf("evidence_round:%s:%s", stage, q.Question)
	prevRounds, _ := evidence["rounds"].(int)
	evidence["rounds"] = prevRounds + 1

	facts := []string{}
	lower := strings.ToLower(p.brief)
	for _, opt := range q.Options {
		if strings.Contains(lower, strings.ToLower(opt)) {
			facts = append(facts, "brief menyebut opsinya: "+opt)
		}
	}
	// Artefak tahap sebelumnya jadi fakta tambahan (digest ringkas).
	for st, c := range p.prevDigest {
		facts = append(facts, fmt.Sprintf("artefak %s: %s", st, truncate(c, 200)))
	}
	if prev, ok := evidence[key]; ok && len(prev.([]string)) >= len(facts) {
		return false // tidak ada fakta baru
	}
	evidence[key] = facts
	return len(facts) != 0
}

// recordDecision menyimpan baris decisions untuk setiap panggilan JEV.
func (p *Pipeline) recordDecision(ctx context.Context, run *entity.PlanRun, stage entity.Stage,
	q GateQuestion, round int, gate entity.Gate, choice string, conf float64,
	resp *port.JudgeResponse, callErr error) {

	d := &entity.Decision{
		ID:       p.ids.New(),
		RunID:    run.ID,
		UserID:   run.UserID,
		Stage:    stage,
		Question: q.Question,
		Gate:     gate,
		Round:    round,
	}
	if choice != "" {
		c := choice
		d.Choice = &c
	}
	if conf > 0 {
		c := conf
		d.Confidence = &c
	}
	if resp != nil {
		d.JEVDecisionID = strPtr(resp.DecisionID)
		if b, err := json.Marshal(resp.State); err == nil {
			d.State = b
		}
		if b, err := json.Marshal(resp.Criteria); err == nil {
			d.Criteria = b
		}
		if a, ok := resp.Answers["q1"]; ok && len(a.Probabilities) > 0 {
			if b, err := json.Marshal(a.Probabilities); err == nil {
				d.Probabilities = b
			}
		}
	}
	if callErr != nil {
		e := callErr.Error()
		d.State, _ = json.Marshal(map[string]string{"error": e})
	}
	// Kegagalan mencatat keputusan tidak boleh menjatuhkan run; dicatat log.
	if err := p.decisions.Create(ctx, d); err != nil {
		p.log.Warn("catat decision gagal", "run", run.ID, "err", err)
	}
	run.Escalations++
}
