package repository

import "paluketok/server/internal/port"

// kompilasi: semua port repo terpenuhi oleh implementasi pgx
var (
	_ port.ProfileRepo     = (*ProfileRepo)(nil)
	_ port.ProjectRepo     = (*ProjectRepo)(nil)
	_ port.RunRepo         = (*RunRepo)(nil)
	_ port.ArtifactRepo    = (*ArtifactRepo)(nil)
	_ port.DecisionRepo    = (*DecisionRepo)(nil)
	_ port.OrderRepo       = (*OrderRepo)(nil)
	_ port.LedgerRepo      = (*LedgerRepo)(nil)
	_ port.MessageRepo     = (*MessageRepo)(nil)
	_ port.ChatSessionRepo = (*ChatSessionRepo)(nil)
	_ port.SkillRepo       = (*SkillRepo)(nil)
)
