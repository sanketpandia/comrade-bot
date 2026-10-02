/**
 * Politburo HTTP facade. New OpenAPI-covered calls must go through
 * `src/generated/politburoClient.ts` (see `getPolitburoClient`). Hand-written
 * fetch below is legacy until those routes are migrated.
 */
import fetch from "node-fetch";
import { 
    HealthApiResponse, 
    InitRegistrationResponse, 
    ApiResponse, 
    FlightHistoryPage, 
    InitServerResponse, 
    LiveFlightRecord, 
    UserDetailsData, 
    PilotStatsData, 
    PirepConfigResponse, 
    PirepSubmitResponse, 
    PirepSubmitRequest, 
    RegistrationResult, 
    MembershipJoinResult, 
    InitServerResult,
    EventsResponse,
    EventResponse,
    TourLegResponse
} from "../types/Responses";
import { MetaInfo } from "../types/DiscordInteraction";
import {
    generateMetaHeaders,
    generateRegistrationMetaHeaders,
    getPolitburoApiUrl,
} from "../helpers/utils";
import { UnauthorizedError } from "../helpers/UnauthorizedException";
import { PermissionDeniedError } from "../helpers/PermissionDeniedException";
import { NotFoundError } from "../helpers/NotFoundException";
import { errorFields, logger } from "../infra/logger";
import { unwrapApiData } from "../helpers/apiEnvelope";
import { ApiNotImplementedError } from "../helpers/ApiNotImplementedError";
import { getPolitburoClient } from "../generated/politburoClient";
import { PolitburoApiError } from "../helpers/PolitburoApiError";

type PolitburoErrorBody = {
    error?: { code?: string; message?: string };
    message?: string;
};

export class ApiService {
    private static rejectStub<T>(operation: string): Promise<T> {
        logger.warn("api_operation_stubbed", { operation });
        return Promise.reject(new ApiNotImplementedError(operation));
    }

    private static async parsePolitburoError(res: Awaited<ReturnType<typeof fetch>>): Promise<{ code?: string; message?: string }> {
        try {
            const body = await res.json() as PolitburoErrorBody;
            return {
                code: body.error?.code,
                message: body.error?.message || body.message,
            };
        } catch {
            return {};
        }
    }

    /** Logs Politburo upstream failures. Uses api_path (not route) for Loki/Grafana filters. */
    private static logUpstreamAPIFailure(
        operation: string,
        method: string,
        apiPath: string,
        status: number,
        errorCode?: string,
        err?: unknown,
    ): void {
        const fields: Record<string, unknown> = {
            operation,
            method,
            api_path: apiPath,
            status,
        };
        if (errorCode) {
            fields.error_code = errorCode;
        }
        if (err) {
            Object.assign(fields, errorFields(err));
        }
        if (status >= 500) {
            logger.error("api_request_failed", fields);
            return;
        }
        logger.warn("api_request_failed", fields);
    }

    static async getHealth(metainfo: MetaInfo): Promise<HealthApiResponse> {
        const controller = new AbortController();
        const timeout = setTimeout(() => controller.abort(), 5000);
        try {
            const res = await fetch(`${getPolitburoApiUrl()}/health/status`, {
                method: "GET",
                headers: generateMetaHeaders(metainfo),
                signal: controller.signal,
            });
            const data = await res.json() as {
                status?: string;
                started_at?: string;
                uptime?: string;
                services?: Record<string, string>;
            };
            if (res.status !== 200 && res.status !== 503) {
                throw new Error(`Failed to fetch health/status: ${res.status} ${res.statusText}`);
            }
            return {
                status: data.status ?? "unknown",
                up_since: data.started_at ?? "",
                uptime: data.uptime ?? "",
                httpStatus: res.status,
                services: data.services ?? {},
            };
        } catch (err) {
            logger.warn("api_request_failed", {
                operation: "get_health",
                ...errorFields(err),
            });
            throw err;
        } finally {
            clearTimeout(timeout);
        }
    }

    static async initiateRegistration(
        meta: MetaInfo,
        ifcId: string,
        lastFlight: string
    ): Promise<RegistrationResult> {
        const apiPath = "/api/v1/users";
        try {
            const client = getPolitburoClient(getPolitburoApiUrl());
            const result = await client.createUser(meta, {
                discourseNames: [ifcId],
                logbookEntry: lastFlight,
            });
            return {
                success: result.success,
                message: result.message,
                is_va_registered: result.is_va_registered,
            };
        } catch (err) {
            if (PolitburoApiError.isPolitburoApiError(err)) {
                this.logUpstreamAPIFailure(
                    "initiate_registration",
                    "POST",
                    apiPath,
                    err.httpStatus,
                    String(err.code),
                    err,
                );
                throw err;
            }
            logger.error("api_request_failed", {
                operation: "initiate_registration",
                method: "POST",
                api_path: apiPath,
                ...errorFields(err),
            });
            throw err;
        }
    }

    // Change the return type so the caller gets the envelope for both 200 and 500

    static async initiateServerRegistration(
        meta: MetaInfo,
        code: string
    ): Promise<InitServerResult> {
        try {
            const res = await fetch(`${getPolitburoApiUrl()}/api/v1/server/init`, {
                method: "POST",
                headers: generateRegistrationMetaHeaders(meta),
                body: JSON.stringify({
                    va_code: code,
                }),
            });

            if (res.status === 401) {
                const message = await res.text(); // plain-text body
                throw new UnauthorizedError(message || "Unauthorized");
            }

            if (res.status === 403) {
                const body = await res.json() as ApiResponse<any>;
                const errorCode = body.error?.code;
                throw new PermissionDeniedError(`${errorCode ? `${errorCode}: ` : ""}${body.error?.message || body.message || "Forbidden"}`);
            }

            if (res.status === 400) {
                const body = await res.json() as any;
                const errorCode = body.error?.code;
                const errorMsg = body.error?.message || body.message || "You must register as a user before initializing a server";
                throw new Error(`${errorCode ? `${errorCode}: ` : ""}${errorMsg}`);
            }

            if (res.status === 409) {
                const body = await res.json() as any;
                const errorCode = body.error?.code;
                const errorMsg = body.error?.message || body.message || "This Discord server is already registered as a VA";
                throw new Error(`${errorCode ? `${errorCode}: ` : ""}${errorMsg}`);
            }

            if (!res.ok) {
                throw new Error(`Failed to initialize server: ${res.status} ${res.statusText}`);
            }

            const response: ApiResponse<InitServerResult> = await res.json() as ApiResponse<InitServerResult>;

            const result = unwrapApiData<InitServerResult>(response as Record<string, unknown>);
            if (!result) {
                throw new Error("No data received in API response");
            }
            return result;
        } catch (err) {
            // network/CORS/JSON issues
            console.error("[ApiService.initiateServerRegistration]", err);
            throw err;
        }
    }


    static async getUserLogbook(meta: MetaInfo, ifcId: string, page: number): Promise<FlightHistoryPage & { response_time: string }> {
        return ApiService.rejectStub(`GET /api/v1/user/${ifcId}/flights?page=${page}`);
    }

    static async getLiveFlights(meta: MetaInfo): Promise<{ flights: LiveFlightRecord[], responseTime?: string, signedLink?: string }> {
        return ApiService.rejectStub("GET /api/v1/flights/va");
    }



    static async getUserDetails(meta: MetaInfo): Promise<UserDetailsData> {
        const apiPath = "/api/v1/user/status";
        try {
            const res = await fetch(`${getPolitburoApiUrl()}${apiPath}`, {
                method: "GET",
                headers: generateRegistrationMetaHeaders(meta),
            });

            if (res.status === 401) {
                const message = await res.text();
                this.logUpstreamAPIFailure("get_user_details", "GET", apiPath, res.status, "UNAUTHORIZED");
                throw new UnauthorizedError(message || "Unauthorized");
            }

            if(res.status === 404) {
                this.logUpstreamAPIFailure("get_user_details", "GET", apiPath, res.status, "NOT_FOUND");
                throw new NotFoundError("User not found");
            }

            if (!res.ok) {
                const { code, message } = await this.parsePolitburoError(res);
                this.logUpstreamAPIFailure("get_user_details", "GET", apiPath, res.status, code);
                throw new Error(message || `Failed to fetch user details: ${res.status} ${res.statusText}`);
            }

            const response: ApiResponse<UserDetailsData> = await res.json() as ApiResponse<UserDetailsData>;

            const result = unwrapApiData<UserDetailsData>(response as Record<string, unknown>);
            if (!result) {
                throw new Error("No data received in API response");
            }
            return result;
        } catch (err) {
            if (
                err instanceof UnauthorizedError ||
                err instanceof NotFoundError ||
                (err instanceof Error && err.message.startsWith("Failed to fetch user details:"))
            ) {
                throw err;
            }
            logger.error("api_request_failed", {
                operation: "get_user_details",
                method: "GET",
                api_path: apiPath,
                ...errorFields(err),
            });
            throw err;
        }
    }

    /**
     * Verifies if the current user has god-mode access
     * Returns true if user is god-mode, false otherwise
     */
    static async verifyGodMode(meta: MetaInfo): Promise<boolean> {
        try {
            const res = await fetch(`${getPolitburoApiUrl()}/api/v1/admin/verify-god`, {
                method: "GET",
                headers: generateRegistrationMetaHeaders(meta),
            });

            if (res.status === 401 || res.status === 403) {
                // Not authorized or forbidden = not god mode
                return false;
            }

            if (!res.ok) {
                console.error("[ApiService.verifyGodMode] Unexpected status:", res.status);
                return false;
            }

            const response = await res.json() as Record<string, unknown>;
            const result = unwrapApiData<{ is_god: boolean }>(response);
            return result?.is_god || false;
        } catch (err) {
            console.error("[ApiService.verifyGodMode]", err);
            return false;
        }
    }

    static async getPilotStats(meta: MetaInfo): Promise<ApiResponse<PilotStatsData>> {
        return ApiService.rejectStub("GET /api/v1/pilot/stats");
    }

    /**
     * Fetch PIREP configuration for the current user's flight
     * Returns available flight modes with validation status and field definitions
     */
    static async getPirepConfig(meta: MetaInfo): Promise<PirepConfigResponse> {
        return ApiService.rejectStub("GET /api/v1/pireps/config");
    }

    /**
     * Submit a PIREP for filing
     * Handles all flight modes with mode-specific validation
     */
    static async submitPirep(meta: MetaInfo, pirepData: PirepSubmitRequest): Promise<PirepSubmitResponse> {
        return ApiService.rejectStub("POST /api/v1/pireps/submit");
    }

    /**
     * Generate a signed link with redirect URL support
     * @param meta - Meta information for API authentication
     * @param redirectTo - URL to redirect to after authentication (can include query parameters)
     * @param ttlMinutes - Time to live in minutes (default: 15)
     * @returns Signed link URL and expiration info
     */
    static async generateSignedLink(
        meta: MetaInfo,
        redirectTo: string,
        ttlMinutes?: number
    ): Promise<ApiResponse<{ url: string; expires_in: number; redirect_to: string }>> {
        try {
            const res = await fetch(`${getPolitburoApiUrl()}/api/v1/signed-link`, {
                method: "POST",
                headers: {
                    ...generateRegistrationMetaHeaders(meta),
                    "Content-Type": "application/json"
                },
                body: JSON.stringify({
                    redirectTo,
                })
            });

            if (res.status === 401) {
                const message = await res.text();
                throw new UnauthorizedError(message || "Unauthorized");
            }

            if (res.status === 403) {
                const body = await res.json() as ApiResponse<any>;
                throw new PermissionDeniedError(body.message || "Forbidden");
            }

            if (!res.ok) {
                throw new Error(`Failed to generate signed link: ${res.status} ${res.statusText}`);
            }

            const raw = await res.json() as Record<string, unknown>;
            const data = unwrapApiData<{ url: string; expiresIn?: number; expires_in?: number; redirectTo?: string; redirect_to?: string }>(raw);
            if (!data?.url) {
                throw new Error("No data received in API response");
            }
            return {
                status: "ok",
                result: {
                    url: data.url,
                    expires_in: data.expiresIn ?? data.expires_in ?? 600,
                    redirect_to: data.redirectTo ?? data.redirect_to ?? redirectTo,
                },
            };
        } catch (err) {
            console.error("[ApiService.generateSignedLink]", err);
            throw err;
        }
    }

    /**
     * Join a virtual airline as a member with a callsign
     * Requires the user to be registered first (will error with USER_NOT_FOUND if not)
     */
    static async joinMembership(meta: MetaInfo, callsign: string): Promise<MembershipJoinResult> {
        try {
            const res = await fetch(`${getPolitburoApiUrl()}/api/v1/memberships/join`, {
                method: "POST",
                headers: {
                    ...generateRegistrationMetaHeaders(meta),
                    "Content-Type": "application/json"
                },
                body: JSON.stringify({ callsign })
            });

            if (res.status === 401) {
                const message = await res.text();
                throw new UnauthorizedError(message || "Unauthorized");
            }

            if (res.status === 403) {
                const body = await res.json() as ApiResponse<any>;
                throw new PermissionDeniedError(body.message || "Forbidden");
            }

            // Handle specific error codes
            if (res.status === 409) {
                const body = await res.json() as any;
                const errorMsg = body.error?.message || body.message;
                if (errorMsg?.includes("already a member") || errorMsg?.includes("ALREADY_MEMBER")) {
                    throw new Error("ALREADY_MEMBER: You are already a member of this VA");
                }
                if (errorMsg?.includes("callsign") || errorMsg?.includes("CALLSIGN_TAKEN")) {
                    throw new Error("CALLSIGN_TAKEN: This callsign is already taken");
                }
                throw new Error(errorMsg || "Conflict error");
            }

            if (res.status === 404) {
                const body = await res.json() as any;
                const errorMsg = body.error?.message || body.message;
                if (errorMsg?.includes("VA not found") || errorMsg?.includes("VA_NOT_FOUND")) {
                    throw new Error("VA_NOT_FOUND: Virtual airline not found");
                }
                if (errorMsg?.includes("User not found") || errorMsg?.includes("USER_NOT_FOUND")) {
                    throw new Error("USER_NOT_FOUND: User not found. Please register first using /register");
                }
                throw new Error(errorMsg || "Not found");
            }

            if (!res.ok) {
                throw new Error(`Failed to join membership: ${res.status} ${res.statusText}`);
            }

            const response: ApiResponse<MembershipJoinResult> = await res.json() as ApiResponse<MembershipJoinResult>;

            const result = unwrapApiData<MembershipJoinResult>(response as Record<string, unknown>);
            if (!result) {
                throw new Error("No data received in API response");
            }
            return result;
        } catch (err) {
            console.error("[ApiService.joinMembership]", err);
            throw err;
        }
    }

    /**
     * Get all active events for the current VA
     * Returns list of active events with their legs
     */
    static async getActiveEvents(meta: MetaInfo): Promise<EventsResponse> {
        return ApiService.rejectStub("GET /api/v1/events");
    }

    /**
     * Get a specific event leg by leg number from an event
     * Returns the leg matching the leg_number
     */
    static async getEventLegByNumber(meta: MetaInfo, eventId: string, legNumber: number): Promise<TourLegResponse> {
        return ApiService.rejectStub(`GET /api/v1/events/${eventId}/legs`);
    }

    /**
     * Update the additional_data field for an event leg
     * Accepts a Record<string, any> that will be merged with existing additional_data
     */
    static async updateEventLegAdditionalData(
        meta: MetaInfo,
        eventId: string,
        legId: string,
        additionalData: Record<string, any>
    ): Promise<TourLegResponse> {
        return ApiService.rejectStub(`PATCH /api/v1/events/${eventId}/legs/${legId}/additional-data`);
    }

    static async reportOccupiedIFC(meta: MetaInfo, claimedIfc: string, note?: string): Promise<{ id: string }> {
        const res = await fetch(`${getPolitburoApiUrl()}/api/v1/reports/occupied-ifc`, {
            method: "POST",
            headers: {
                ...generateRegistrationMetaHeaders(meta),
                "Content-Type": "application/json",
            },
            body: JSON.stringify({ claimedIfc, note: note ?? "" }),
        });
        if (res.status === 401) {
            throw new UnauthorizedError(await res.text() || "Unauthorized");
        }
        if (!res.ok) {
            const body = await res.json().catch(() => ({})) as { error?: { message?: string } };
            throw new Error(body.error?.message || `Report failed: ${res.status}`);
        }
        const raw = await res.json() as Record<string, unknown>;
        const data = unwrapApiData<{ id: string }>(raw);
        if (!data?.id) {
            throw new Error("No report id returned");
        }
        return data;
    }
}
