// SPDX-License-Identifier: 0BSD

package acp

// PlanEntryPriority indicates relative importance of a plan entry.
type PlanEntryPriority string

// Plan entry priorities.
const (
	PriorityHigh   PlanEntryPriority = "high"
	PriorityMedium PlanEntryPriority = "medium"
	PriorityLow    PlanEntryPriority = "low"
)

// PlanEntryStatus tracks an entry through the execution flow.
type PlanEntryStatus string

// Plan entry statuses.
const (
	PlanPending    PlanEntryStatus = "pending"
	PlanInProgress PlanEntryStatus = "in_progress"
	PlanCompleted  PlanEntryStatus = "completed"
)

// PlanEntry is one task in the agent's execution plan.
type PlanEntry struct {
	Content  string            `json:"content"`
	Priority PlanEntryPriority `json:"priority"`
	Status   PlanEntryStatus   `json:"status"`
	Meta     Meta              `json:"_meta,omitempty"`
}

// Plan is the agent's execution plan for a complex task, reported via
// session updates.
type Plan struct {
	Entries []PlanEntry `json:"entries"`
	Meta    Meta        `json:"_meta,omitempty"`
}
