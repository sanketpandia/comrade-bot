package identity

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"slices"
	"strings"

	"github.com/go-chi/chi/v5"

	"infinite-experiment/politburo/internal/access/auth"
	"infinite-experiment/politburo/internal/community/membership"
	"infinite-experiment/politburo/internal/operations/operator"
	"infinite-experiment/politburo/internal/community/registration"
	"infinite-experiment/politburo/internal/operations/reports"
	"infinite-experiment/politburo/internal/transport/http/response"
	"infinite-experiment/politburo/internal/community/users/status"
	"infinite-experiment/politburo/internal/community/virtualairlines"
)

type Handler struct {
	registration *registration.Service
	membership   *membership.Service
	vaInit       *virtualairlines.Service
	status       *status.Builder
	reports      *reports.Repository
	operator     *operator.Service
	vas          *virtualairlines.Repository
	operators    []string
}

func NewHandler(
	registration *registration.Service,
	membership *membership.Service,
	vaInit *virtualairlines.Service,
	status *status.Builder,
	reports *reports.Repository,
	operator *operator.Service,
	vas *virtualairlines.Repository,
	platformOperators []string,
) *Handler {
	return &Handler{
		registration: registration,
		membership:   membership,
		vaInit:       vaInit,
		status:       status,
		reports:      reports,
		operator:     operator,
		vas:          vas,
		operators:    platformOperators,
	}
}

func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.ClaimsFromContext(r.Context())
	if !ok {
		response.WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Unauthorized")
		return
	}
	var body registerBody
	if err := decodeJSON(r, &body); err != nil {
		response.WriteError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
		return
	}
	ifc := firstNonEmpty(body.IFCUsername, body.IFCID)
	proof := firstNonEmpty(body.RouteProof, body.LastFlight)

	result, err := h.registration.Register(r.Context(), registration.RegisterInput{
		DiscordID:   claims.DsUserID,
		IFCUsername: ifc,
		RouteProof:  proof,
	})
	if err != nil {
		writeRegisterError(w, err)
		return
	}

	isVARegistered := false
	if claims.DsServerID != "" {
		va, _ := h.vas.GetByDiscordServerID(r.Context(), claims.DsServerID)
		isVARegistered = va != nil
	}

	response.WriteJSON(w, http.StatusOK, map[string]any{
		"data": map[string]any{
			"success":           true,
			"message":           "Registration successful",
			"is_va_registered":  isVARegistered,
			"if_community_id":   result.User.IFCommunityID,
		},
	})
}

func (h *Handler) Status(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.ClaimsFromContext(r.Context())
	if !ok {
		response.WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Unauthorized")
		return
	}
	status, err := h.status.Build(r.Context(), claims.DsUserID, claims.DsServerID)
	if err != nil {
		response.WriteError(w, http.StatusInternalServerError, "STATUS_FAILED", "failed to build user status")
		return
	}
	response.WriteJSON(w, http.StatusOK, map[string]any{"data": status})
}

func (h *Handler) Join(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.ClaimsFromContext(r.Context())
	if !ok {
		response.WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Unauthorized")
		return
	}
	var body joinBody
	if err := decodeJSON(r, &body); err != nil {
		response.WriteError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
		return
	}
	result, err := h.membership.JoinCurrentServer(r.Context(), membership.JoinCurrentServerInput{
		DiscordID:       claims.DsUserID,
		DiscordServerID: claims.DsServerID,
		Callsign:        body.Callsign,
	})
	if err != nil {
		writeJoinError(w, err)
		return
	}
	callsign := body.Callsign
	if result.Membership.Callsign != nil {
		callsign = *result.Membership.Callsign
	}
	response.WriteJSON(w, http.StatusOK, map[string]any{
		"data": map[string]any{
			"success":  true,
			"callsign": callsign,
			"role":     result.Membership.Role,
			"va_id":    result.Membership.VAID,
		},
	})
}

func (h *Handler) InitServer(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.ClaimsFromContext(r.Context())
	if !ok {
		response.WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Unauthorized")
		return
	}
	var body initServerBody
	if err := decodeJSON(r, &body); err != nil {
		response.WriteError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
		return
	}
	code := firstNonEmpty(body.VACode, body.VaCode)
	result, err := h.vaInit.InitServer(r.Context(), virtualairlines.InitInput{
		DiscordUserID:   claims.DsUserID,
		DiscordServerID: claims.DsServerID,
		VACode:          code,
		DisplayName:     body.DisplayName,
	})
	if err != nil {
		writeInitServerError(w, err)
		return
	}
	response.WriteJSON(w, http.StatusOK, map[string]any{
		"data": map[string]any{
			"success":         true,
			"message":         "Virtual airline initialized",
			"va_code":         result.VA.Code,
			"setup_required":  result.SetupRequired,
			"va_id":           result.VA.ID,
		},
	})
}

func (h *Handler) ReportOccupiedIFC(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.ClaimsFromContext(r.Context())
	if !ok {
		response.WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Unauthorized")
		return
	}
	var body occupiedIFCBody
	if err := decodeJSON(r, &body); err != nil {
		response.WriteError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
		return
	}
	claimed := firstNonEmpty(body.ClaimedIFC, body.ClaimedIfc)
	report, err := h.reports.CreateOccupiedIFC(r.Context(), claims.DsUserID, claimed, body.Note)
	if err != nil {
		response.WriteError(w, http.StatusInternalServerError, "REPORT_FAILED", "failed to create report")
		return
	}
	response.WriteJSON(w, http.StatusCreated, map[string]any{"data": map[string]any{"id": report.ID}})
}

func (h *Handler) ListOperatorReports(w http.ResponseWriter, r *http.Request) {
	reports, err := h.reports.ListOpen(r.Context())
	if err != nil {
		response.WriteError(w, http.StatusInternalServerError, "LIST_FAILED", "failed to list reports")
		return
	}
	response.WriteJSON(w, http.StatusOK, map[string]any{"data": reports})
}

func (h *Handler) ResolveOperatorReport(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.ClaimsFromContext(r.Context())
	if !ok {
		response.WriteError(w, http.StatusForbidden, "FORBIDDEN", "platform operator access required")
		return
	}
	reportID := chi.URLParam(r, "reportID")
	if reportID == "" {
		response.WriteError(w, http.StatusBadRequest, "INVALID_REQUEST", "report id required")
		return
	}
	report, err := h.reports.GetByID(r.Context(), reportID)
	if err != nil || report == nil {
		response.WriteError(w, http.StatusNotFound, "NOT_FOUND", "report not found")
		return
	}
	switch report.Kind {
	case reports.KindOccupiedIFC:
		err = h.operator.ResolveOccupiedIFC(r.Context(), reportID, claims.DsUserID)
	case reports.KindMigrateDiscordServer:
		err = h.operator.ResolveGuildMigration(r.Context(), reportID, claims.DsUserID)
	default:
		response.WriteError(w, http.StatusBadRequest, "INVALID_REQUEST", "unsupported report kind")
		return
	}
	if err != nil {
		if errors.Is(err, operator.ErrOccupantNotFound) {
			response.WriteError(w, http.StatusNotFound, "OCCUPANT_NOT_FOUND", "no user linked to claimed IFC")
			return
		}
		response.WriteError(w, http.StatusInternalServerError, "RESOLVE_FAILED", "failed to resolve report")
		return
	}
	response.WriteJSON(w, http.StatusOK, map[string]any{"data": map[string]any{"resolved": true}})
}

func (h *Handler) VerifyGod(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.ClaimsFromContext(r.Context())
	if !ok {
		response.WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Unauthorized")
		return
	}
	isGod := slices.Contains(h.operators, claims.DsUserID)
	if !isGod {
		response.WriteError(w, http.StatusForbidden, "FORBIDDEN", "god mode required")
		return
	}
	response.WriteJSON(w, http.StatusOK, map[string]any{
		"data": map[string]any{"is_god": true},
	})
}

type registerBody struct {
	IFCUsername string `json:"ifcUsername"`
	IFCID       string `json:"ifc_id"`
	RouteProof  string `json:"routeProof"`
	LastFlight  string `json:"last_flight"`
}

type joinBody struct {
	Callsign string `json:"callsign"`
}

type initServerBody struct {
	VACode      string `json:"va_code"`
	VaCode      string `json:"vaCode"`
	DisplayName string `json:"display_name"`
}

type occupiedIFCBody struct {
	ClaimedIFC string `json:"claimedIfc"`
	ClaimedIfc string `json:"claimed_ifc"`
	Note       string `json:"note"`
}

func decodeJSON(r *http.Request, dest any) error {
	if r.Body == nil {
		return nil
	}
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(dest); err != nil && err != io.EOF {
		return err
	}
	return nil
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

func writeRegisterError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, registration.ErrBanned):
		response.WriteError(w, http.StatusForbidden, "BANNED", "This Discord account cannot register")
	case errors.Is(err, registration.ErrAlreadyRegistered):
		response.WriteError(w, http.StatusConflict, "USER_ALREADY_REGISTERED", "User already registered")
	case errors.Is(err, registration.ErrIFCAlreadyLinked):
		response.WriteError(w, http.StatusConflict, "IFC_ALREADY_LINKED", "IFC ID is already registered to another Discord account")
	case errors.Is(err, registration.ErrIFUserNotFound):
		response.WriteError(w, http.StatusNotFound, "IF_USER_NOT_FOUND", "IFC user not found")
	case errors.Is(err, registration.ErrFlightProofFailed):
		response.WriteError(w, http.StatusBadRequest, "FLIGHT_PROOF_FAILED", "Flight validation failed")
	case errors.Is(err, registration.ErrIFClientUnavailable):
		response.WriteError(w, http.StatusServiceUnavailable, "IF_UNAVAILABLE", "Infinite Flight API is not configured")
	default:
		response.WriteError(w, http.StatusBadRequest, "REGISTRATION_FAILED", err.Error())
	}
}

func writeJoinError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, membership.ErrUserNotFound):
		response.WriteError(w, http.StatusNotFound, "USER_NOT_FOUND", "User not found. Please register first")
	case errors.Is(err, membership.ErrVANotFound):
		response.WriteError(w, http.StatusNotFound, "VA_NOT_FOUND", "Virtual airline not found for this server")
	case errors.Is(err, membership.ErrAlreadyMember):
		response.WriteError(w, http.StatusConflict, "ALREADY_MEMBER", "You are already a member of this VA")
	case errors.Is(err, membership.ErrCallsignRequired):
		response.WriteError(w, http.StatusBadRequest, "CALLSIGN_REQUIRED", "Callsign cannot be empty")
	default:
		response.WriteError(w, http.StatusBadRequest, "JOIN_FAILED", err.Error())
	}
}

func writeInitServerError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, virtualairlines.ErrUserNotFound):
		response.WriteError(w, http.StatusBadRequest, "USER_NOT_FOUND", "You must register as a user before initializing a server")
	case errors.Is(err, virtualairlines.ErrServerAlreadyVA):
		response.WriteError(w, http.StatusConflict, "SERVER_ALREADY_VA", "This Discord server is already registered as a VA")
	case errors.Is(err, virtualairlines.ErrCodeTaken):
		response.WriteError(w, http.StatusConflict, "VA_CODE_TAKEN", "This VA code is already in use")
	case errors.Is(err, virtualairlines.ErrInvalidVACode):
		response.WriteError(w, http.StatusBadRequest, "INVALID_VA_CODE", "Invalid VA code")
	default:
		response.WriteError(w, http.StatusInternalServerError, "INIT_FAILED", err.Error())
	}
}
