package featureconfig

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"strings"

	"infinite-experiment/politburo/internal/community/virtualairlines/connections"
)

type recordReader interface {
	List(ctx context.Context, vaID, entity string, filters map[string]any, pageSize int, cursor string) (connections.ListResult, error)
	Get(ctx context.Context, vaID, entity, sourceID string) (connections.MappedRecord, error)
	GetView(ctx context.Context, vaID string) (connections.StoredView, error)
}

// Previewer resolves stats and filing previews.
type Previewer struct {
	reader recordReader
}

func NewPreviewer(reader recordReader) *Previewer {
	return &Previewer{reader: reader}
}

func (p *Previewer) Stats(ctx context.Context, vaID string, doc StatsDocument, callsign, ifcUsername string) (PreviewResult, error) {
	out := PreviewResult{}
	if doc.Pilot != nil {
		sec, warns, err := p.section(ctx, vaID, connections.EntityPilots, doc.Pilot, callsign, ifcUsername, true)
		if err != nil {
			if errors.Is(err, ErrPilotNotFound) || errors.Is(err, ErrAmbiguousPilot) {
				return PreviewResult{}, err
			}
			return PreviewResult{}, err
		}
		out.Sections = append(out.Sections, sec)
		out.Warnings = append(out.Warnings, warns...)
	}
	if doc.Career != nil {
		sec, warns, err := p.section(ctx, vaID, connections.EntityCareerPilots, doc.Career, callsign, ifcUsername, false)
		if err != nil {
			var pe *connections.ProviderError
			if errors.As(err, &pe) || errors.Is(err, ErrPilotNotFound) {
				out.Warnings = append(out.Warnings, err.Error())
				sec = PreviewSection{Name: "Career", Fields: []PreviewField{}}
			} else {
				return PreviewResult{}, err
			}
		}
		out.Sections = append(out.Sections, sec)
		out.Warnings = append(out.Warnings, warns...)
	}
	return out, nil
}

func (p *Previewer) section(ctx context.Context, vaID, entity string, sec *StatsSection, callsign, ifcUsername string, required bool) (PreviewSection, []string, error) {
	name := "Pilot"
	if entity == connections.EntityCareerPilots {
		name = "Career"
	}
	key := sec.MatchBy
	var identifier string
	switch key {
	case "callsign":
		identifier = strings.TrimSpace(callsign)
	case "ifcUsername":
		identifier = strings.TrimSpace(ifcUsername)
	default:
		return PreviewSection{Name: name}, nil, fmt.Errorf("%w: invalid matchBy", ErrInvalidDraft)
	}
	if identifier == "" {
		slog.Warn("stats preview missing test identity",
			"component", "feature_mappings",
			"va_id", vaID,
			"section", name,
			"match_by", key,
		)
		return PreviewSection{Name: name}, nil, fmt.Errorf("%w: test identity required", ErrInvalidDraft)
	}
	if entity == connections.EntityCareerPilots && key == "ifcUsername" {
		view, viewErr := p.reader.GetView(ctx, vaID)
		if viewErr != nil {
			return PreviewSection{}, nil, viewErr
		}
		careerMap := view.Mappings[connections.EntityCareerPilots]
		if _, direct := careerMap.Fields["ifcUsername"]; !direct && careerIFCViaRosterAllowed(view.Mappings) {
			return p.sectionCareerMatchedByIFCViaRoster(ctx, vaID, sec, identifier, required)
		}
	}
	filterVal := identifier
	if key == "callsign" {
		filterVal = connections.NormalizeCallsignForMatch(identifier)
	}
	filters := map[string]any{key: filterVal}
	page, err := p.reader.List(ctx, vaID, entity, filters, 2, "")
	if err != nil {
		return PreviewSection{}, nil, err
	}
	var warnings []string
	if len(page.Records) == 0 {
		if required {
			return PreviewSection{}, nil, ErrPilotNotFound
		}
		return PreviewSection{Name: name}, warnings, ErrPilotNotFound
	}
	if len(page.Records) > 1 {
		if required {
			warnings = append(warnings, "ambiguous pilot match")
			return PreviewSection{}, warnings, ErrAmbiguousPilot
		}
		warnings = append(warnings, "ambiguous pilot match")
	}
	rec := page.Records[0]
	fields := make([]PreviewField, 0, len(sec.Fields))
	for _, logicalKey := range sec.Fields {
		cf, _ := connections.CatalogFieldByKey(entity, logicalKey)
		if logicalKey == "lastFlownRoute" {
			val, warn, err := p.resolveLastFlownRoute(ctx, vaID, rec)
			if err != nil {
				warnings = append(warnings, err.Error())
			}
			fields = append(fields, PreviewField{Key: logicalKey, Label: "Last flown route", Value: val, Display: formatAny(val), Warning: warn})
			continue
		}
		val, ok := rec.Fields[logicalKey]
		pf := PreviewField{Key: logicalKey, Label: cf.Label, Unresolved: !ok}
		if ok {
			pf.Value = val
			pf.Display = formatField(cf.Kind, val)
		}
		fields = append(fields, pf)
	}
	return PreviewSection{Name: name, Fields: fields}, warnings, nil
}

func (p *Previewer) sectionCareerMatchedByIFCViaRoster(ctx context.Context, vaID string, sec *StatsSection, ifcUsername string, required bool) (PreviewSection, []string, error) {
	page, err := p.reader.List(ctx, vaID, connections.EntityPilots, map[string]any{"ifcUsername": strings.TrimSpace(ifcUsername)}, 2, "")
	if err != nil {
		return PreviewSection{}, nil, err
	}
	if len(page.Records) == 0 {
		if required {
			return PreviewSection{}, nil, ErrPilotNotFound
		}
		return PreviewSection{Name: "Career"}, nil, ErrPilotNotFound
	}
	if len(page.Records) > 1 {
		return PreviewSection{}, []string{"ambiguous pilot match"}, ErrAmbiguousPilot
	}
	rawCS, ok := page.Records[0].Fields["callsign"]
	if !ok {
		return PreviewSection{}, nil, fmt.Errorf("%w: roster row has no callsign for ifcUsername", ErrPilotNotFound)
	}
	callsign, ok := rawCS.(string)
	if !ok || strings.TrimSpace(callsign) == "" {
		return PreviewSection{}, nil, fmt.Errorf("%w: roster row has no callsign for ifcUsername", ErrPilotNotFound)
	}
	slog.Info("stats career match via roster callsign",
		"component", "feature_mappings",
		"va_id", vaID,
		"ifc_username", strings.TrimSpace(ifcUsername),
		"resolved_callsign", callsign,
	)
	careerSec := *sec
	careerSec.MatchBy = "callsign"
	return p.section(ctx, vaID, connections.EntityCareerPilots, &careerSec, callsign, "", required)
}

func (p *Previewer) resolveLastFlownRoute(ctx context.Context, vaID string, careerRec connections.MappedRecord) (any, string, error) {
	raw, ok := careerRec.Fields["lastLogRefs"]
	if !ok {
		return nil, "", nil
	}
	ids, ok := raw.([]string)
	if !ok {
		if s, ok := raw.(string); ok && s != "" {
			ids = []string{s}
		} else {
			return nil, "", nil
		}
	}
	if len(ids) == 0 {
		return nil, "", nil
	}
	if len(ids) > 1 {
		return nil, "multiple last-log links; cannot determine latest", nil
	}
	pirep, err := p.reader.Get(ctx, vaID, connections.EntityPIREPs, ids[0])
	if err != nil {
		return nil, "", err
	}
	if label, ok := pirep.Fields["routeLabel"].(string); ok && label != "" {
		return label, "", nil
	}
	routeRefs, ok := pirep.Fields["routeRefs"].([]string)
	if !ok || len(routeRefs) == 0 {
		return nil, "", nil
	}
	route, err := p.reader.Get(ctx, vaID, connections.EntityRoutes, routeRefs[0])
	if err != nil {
		return nil, "", err
	}
	if label, ok := route.Fields["routeLabel"].(string); ok && label != "" {
		return label, "", nil
	}
	origin, _ := route.Fields["origin"].(string)
	dest, _ := route.Fields["destination"].(string)
	if origin != "" && dest != "" {
		return origin + "–" + dest, "", nil
	}
	return nil, "", nil
}

func (p *Previewer) PIREP(ctx context.Context, vaID string, doc PIREPDocument) (PreviewResult, error) {
	view, err := p.reader.GetView(ctx, vaID)
	if err != nil {
		return PreviewResult{}, err
	}
	mapping := view.Mappings[connections.EntityPIREPs]
	out := PreviewResult{}
	for _, key := range doc.Fields {
		cf, _ := connections.CatalogFieldByKey(connections.EntityPIREPs, key)
		fieldID := mapping.Fields[key]
		sample := FilingSample{
			Key: key, Label: cf.Label, Destination: fieldID, Synthetic: true,
		}
		switch key {
		case "flightTimeSeconds":
			sample.Display = "1h 05m"
			sample.WriteValue = sampleWriteDuration(mapping, key, 3900)
		case "pilotRefs":
			sample.Display = "NX042"
		case "routeRefs", "routeLabel":
			sample.Display = "ABC–DEF"
		case "remarks":
			sample.Display = "Sample filing"
			sample.WriteValue = "Sample filing"
		case "passengerCount":
			sample.WriteValue = float64(0)
			sample.Display = "0"
		default:
			sample.Display = "Sample"
		}
		out.Filing = append(out.Filing, sample)
	}
	return out, nil
}

func sampleWriteDuration(mapping connections.Mapping, key string, seconds int64) any {
	unit := mapping.FieldUnits[key]
	switch unit {
	case connections.UnitMinutes:
		return float64(seconds / 60)
	case connections.UnitHours:
		return float64(seconds) / 3600
	default:
		return float64(seconds)
	}
}

func formatAny(v any) string {
	if v == nil {
		return ""
	}
	return displayValue(connections.KindText, v)
}

func formatField(kind connections.FieldKind, v any) string {
	return displayValue(kind, v)
}

func displayValue(kind connections.FieldKind, v any) string {
	switch kind {
	case connections.KindDuration:
		sec, ok := toInt64(v)
		if !ok {
			return fmt.Sprint(v)
		}
		h := sec / 3600
		m := (sec % 3600) / 60
		return fmt.Sprintf("%dh %02dm", h, m)
	default:
		switch t := v.(type) {
		case []string:
			if len(t) > 6 {
				return strings.Join(t[:6], ", ") + fmt.Sprintf(" (%d total)", len(t))
			}
			return strings.Join(t, ", ")
		case string:
			return t
		case float64:
			if math.Mod(t, 1) == 0 {
				return fmt.Sprintf("%.0f", t)
			}
			return fmt.Sprintf("%v", t)
		default:
			return fmt.Sprint(v)
		}
	}
}

func toInt64(v any) (int64, bool) {
	switch t := v.(type) {
	case int64:
		return t, true
	case int:
		return int64(t), true
	case float64:
		return int64(t), true
	default:
		return 0, false
	}
}