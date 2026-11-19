package va

import (
	"errors"
	"log/slog"
	"strings"

	"infinite-experiment/politburo/internal/community/virtualairlines/config"
	"infinite-experiment/politburo/internal/community/virtualairlines/connections"
	"infinite-experiment/politburo/internal/community/virtualairlines/featureconfig"
	"infinite-experiment/politburo/internal/community/virtualairlines/pirepvaluemappings"
)

func logFeatureMappingsFailure(vaID, feature, operation string, err error, preview *featureconfig.PreviewInput) {
	if err == nil {
		return
	}
	args := []any{
		"component", "feature_mappings",
		"va_id", vaID,
		"feature", feature,
		"operation", operation,
		"error_kind", featureMappingsErrorKind(err),
		"error", err.Error(),
	}
	if preview != nil {
		args = append(args,
			"preview_has_callsign", strings.TrimSpace(preview.Callsign) != "",
			"preview_has_ifc_username", strings.TrimSpace(preview.IFCUsername) != "",
		)
		if preview.Stats != nil {
			if preview.Stats.Pilot != nil {
				args = append(args, "stats_pilot_enabled", true, "stats_pilot_match_by", preview.Stats.Pilot.MatchBy)
			}
			if preview.Stats.Career != nil {
				args = append(args, "stats_career_enabled", true, "stats_career_match_by", preview.Stats.Career.MatchBy)
			}
		}
		if preview.PIREP != nil {
			args = append(args, "pirep_field_count", len(preview.PIREP.Fields))
		}
	}
	slog.Warn("feature mapping request failed", args...)
}

func featureMappingsErrorKind(err error) string {
	switch {
	case errors.Is(err, featureconfig.ErrInvalidDraft):
		return "invalid_draft"
	case errors.Is(err, featureconfig.ErrInvalidFeature):
		return "invalid_feature"
	case errors.Is(err, featureconfig.ErrConflict):
		return "conflict"
	case errors.Is(err, featureconfig.ErrStaleConnections):
		return "stale_connections"
	case errors.Is(err, featureconfig.ErrPilotNotFound):
		return "pilot_not_found"
	case errors.Is(err, featureconfig.ErrAmbiguousPilot):
		return "ambiguous_pilot"
	case errors.Is(err, connections.ErrDependency):
		return "feature_dependency"
	case errors.Is(err, config.ErrNotFound):
		return "not_found"
	case errors.Is(err, connections.ErrDisabled):
		return "disabled"
	case errors.Is(err, connections.ErrMissingMapping):
		return "missing_mapping"
	default:
		var pe *connections.ProviderError
		if errors.As(err, &pe) {
			return "provider_error"
		}
		return "other"
	}
}

func logPIREPValueMappingsFailure(vaID, operation string, err error) {
	if err == nil {
		return
	}
	kind := "other"
	switch {
	case errors.Is(err, pirepvaluemappings.ErrConflict):
		kind = "conflict"
	case errors.Is(err, pirepvaluemappings.ErrInvalid):
		kind = "invalid_config"
	case errors.Is(err, pirepvaluemappings.ErrPayloadLarge):
		kind = "payload_large"
	}
	slog.Warn("pirep value mappings request failed",
		"component", "pirep_value_mappings",
		"va_id", vaID,
		"operation", operation,
		"error_kind", kind,
		"error", err.Error(),
	)
}