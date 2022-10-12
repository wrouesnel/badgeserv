package server

import "time"

// LivenessResponse is a common type for responding to K8S style liveness checks.
type LivenessResponse struct {
	RespondedAt time.Time `json:"responded_at"`
}

// ReadinessResponse is a common type for responding to K8S style readiness checks.
type ReadinessResponse struct {
	RespondedAt time.Time `json:"responded_at"`
}

// StartedResonse is a common type for responding to K8S style startup checks.
type StartedResponse struct {
	RespondedAt time.Time `json:"responded_at"`
}
