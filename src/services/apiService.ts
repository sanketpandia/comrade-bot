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
import { generateMetaHeaders } from "../helpers/utils";
import { UnauthorizedError } from "../helpers/UnauthorizedException";
import { PermissionDeniedError } from "../helpers/PermissionDeniedException";
import { NotFoundError } from "../helpers/NotFoundException";

const API_URL = process.env.API_URL ?? "http://localhost:8080";

export class ApiService {
    static async getHealth(metainfo: MetaInfo): Promise<HealthApiResponse> {
        try {
            const res = await fetch(`${API_URL}/healthCheck`, {
                method: "GET",
                headers: generateMetaHeaders(metainfo)
            });
            if (!res.ok) {
                throw new Error(`Failed to fetch healthCheck: ${res.status} ${res.statusText}`);
            }
            const data = await res.json() as HealthApiResponse;
            return data;
        } catch (err) {
            console.error("[ApiService.getHealth]", err);
            throw err;
        }
    }

    static async initiateRegistration(
        meta: MetaInfo,
        ifcId: string,
        lastFlight: string
    ): Promise<RegistrationResult> {
        try {
            const payload = {
                ifc_id: ifcId,
                last_flight: lastFlight
            };

            const res = await fetch(`${API_URL}/api/v1/pilots/register`, {
                method: "POST",
                headers: {
                    ...generateMetaHeaders(meta),
                    "Content-Type": "application/json"
                },
                body: JSON.stringify(payload)
            });

            if (res.status === 401) {
                const message = await res.text(); // plain-text body
                throw new UnauthorizedError(message || "Unauthorized");
            }

            if (res.status === 403) {
                const body = await res.json() as ApiResponse<any>;
                throw new PermissionDeniedError(body.message || "Forbidden");
            }

            if (res.status === 409) {
                const body = await res.json() as any;
                const errorCode = body.error?.code || body.error?.error_code;
                const errorMessage = body.error?.message || body.message;
                
                // Check if it's IFC ID duplicate error
                if (errorCode === "IFC_ID_ALREADY_REGISTERED" || errorMessage?.includes("IFC ID is already registered")) {
                    throw new Error("IFC_ID_ALREADY_REGISTERED: " + (errorMessage || "This IFC ID is already registered to another Discord account."));
                }
                
                // Otherwise, it's the user already registered error
                throw new Error(errorMessage || "User already registered");
            }

            if (res.status === 404) {
                const body = await res.json() as any;
                throw new Error(body.error?.message || "IFC user not found");
            }

            if (res.status === 400) {
                const body = await res.json() as any;
                throw new Error(body.error?.message || "Flight validation failed");
            }

            if (!res.ok) {
                const errorText = await res.text();
                console.error("[ApiService.initiateRegistration] Error response:", res.status, errorText);
                throw new Error(`Failed to fetch initRegistration: ${res.status} ${res.statusText}`);
            }
            
            const responseText = await res.text();
            console.log("[ApiService.initiateRegistration] Raw response:", responseText);
            
            let response: ApiResponse<RegistrationResult>;
            try {
                response = JSON.parse(responseText) as ApiResponse<RegistrationResult>;
            } catch (jsonErr) {
                console.error("[ApiService.initiateRegistration] JSON parse error:", jsonErr);
                throw new Error("Failed to parse API response as JSON");
            }

            // Log response for debugging
            console.log("[ApiService.initiateRegistration] Response:", JSON.stringify(response, null, 2));

            if (!response) {
                throw new Error("Empty response from API");
            }

            if (!response.result) {
                console.error("[ApiService.initiateRegistration] Response missing result field:", JSON.stringify(response, null, 2));
                throw new Error("No data received in API response - result field is missing");
            }
            
            return response.result;
        } catch (err) {
            console.error("[ApiService.initRegistation]", err);
            throw err
        }
    }

    // Change the return type so the caller gets the envelope for both 200 and 500

    static async initiateServerRegistration(
        meta: MetaInfo,
        code: string,
        name: string,
        callsignPrefix: string,
        callsignSuffix: string
    ): Promise<InitServerResult> {
        try {
            const res = await fetch(`${API_URL}/api/v1/server/init`, {
                method: "POST",
                headers: generateMetaHeaders(meta),
                body: JSON.stringify({
                    va_code: code,
                    va_name: name,
                    callsign_prefix: callsignPrefix,
                    callsign_suffix: callsignSuffix
                }),
            });

            if (res.status === 401) {
                const message = await res.text(); // plain-text body
                throw new UnauthorizedError(message || "Unauthorized");
            }

            if (res.status === 403) {
                const body = await res.json() as ApiResponse<any>;
                throw new PermissionDeniedError(body.message || "Forbidden");
            }

            if (res.status === 400) {
                const body = await res.json() as any;
                const errorMsg = body.error?.message || body.message || "You must register as a user before initializing a server";
                throw new Error(errorMsg);
            }

            if (res.status === 409) {
                const body = await res.json() as any;
                const errorMsg = body.error?.message || body.message || "This Discord server is already registered as a VA";
                throw new Error(errorMsg);
            }

            if (!res.ok) {
                throw new Error(`Failed to initialize server: ${res.status} ${res.statusText}`);
            }

            const response: ApiResponse<InitServerResult> = await res.json() as ApiResponse<InitServerResult>;

            if (!response.result) {
                throw new Error("No data received in API response");
            }

            return response.result;
        } catch (err) {
            // network/CORS/JSON issues
            console.error("[ApiService.initiateServerRegistration]", err);
            throw err;
        }
    }


    static async getUserLogbook(meta: MetaInfo, ifcId: string, page: number): Promise<FlightHistoryPage & { response_time: string }> {
        try {
            const res = await fetch(`${API_URL}/api/v1/user/${ifcId}/flights?page=${page}`, {
                method: "GET",
                headers: generateMetaHeaders(meta),
            });
            if (res.status === 401) {
                const message = await res.text(); // plain-text body
                throw new UnauthorizedError(message || "Unauthorized");
            }

            if (!res.ok) {
                throw new Error(`Failed to fetch initRegistration: ${res.status} ${res.statusText}`);
            }


            const response: ApiResponse<FlightHistoryPage> = await res.json() as ApiResponse<FlightHistoryPage>;

            if (!response.result) {
                throw new Error("No data received in API response");
            }

            // Include the responseTimeMs from the API response
            return {
                ...response.result,
                response_time: response.responseTimeMs?.toString() || "0"
            };


        } catch (err) {
            console.error("[ApiService.getLogbook]", err);
            throw err
        }
    }

    static async getLiveFlights(meta: MetaInfo): Promise<{ flights: LiveFlightRecord[], responseTime?: string, signedLink?: string }> {
        try {
            const res = await fetch(`${API_URL}/api/v1/flights/va`, {
                method: "GET",
                headers: generateMetaHeaders(meta),
            });
            if (res.status === 401) {
                const message = await res.text(); // plain-text body
                throw new UnauthorizedError(message || "Unauthorized");
            }

            if (!res.ok) {
                throw new Error(`Failed to fetch live flights: ${res.status} ${res.statusText}`);
            }

            const response = await res.json() as {
                status: string;
                message?: string;
                response_time?: string;
                data?: {
                    flights?: LiveFlightRecord[];
                    signed_link?: string;
                } | LiveFlightRecord[];
                result?: LiveFlightRecord[];
            };

            // Handle both 'data' and 'result' fields for compatibility
            // Check if data is an object with flights and signed_link, or just an array
            let flights: LiveFlightRecord[] | undefined;
            let signedLink: string | undefined;

            if (Array.isArray(response.data)) {
                flights = response.data;
            } else if (response.data && typeof response.data === 'object' && 'flights' in response.data) {
                flights = response.data.flights;
                signedLink = response.data.signed_link;
            } else {
                flights = response.result;
            }

            if (!flights) {
                throw new Error("No data received in API response");
            }

            return {
                flights,
                responseTime: response.response_time,
                signedLink
            };

        } catch (err) {
            console.error("[ApiService.getLiveFlights]", err);
            throw err;
        }
    }



    static async syncUserToVA(meta: MetaInfo, payload: SyncUserPayload): Promise<SyncUserResult> {
        try {
            const res = await fetch(`${API_URL}/api/v1/va/userSync`, {
                method: "POST",
                headers: {
                    ...generateMetaHeaders(meta),
                    "Content-Type": "application/json",
                },
                body: JSON.stringify(payload),
            });

            const body = (await res.json()) as SyncUserResult;
            return body;
        } catch (err) {
            console.error("[ApiService.syncUserToVA]", err);
            throw err;
        }
    }

    static async assignUserRole(meta: MetaInfo, payload: { user_id: string; role: string }): Promise<SyncUserResult> {
        try {
            const res = await fetch(`${API_URL}/api/v1/va/setRole`, {
                method: "POST",
                headers: {
                    ...generateMetaHeaders(meta),
                    "Content-Type": "application/json",
                },
                body: JSON.stringify(payload),
            });
            console.log(payload, "\n", `${API_URL}/api/v1/va/setRole`, "\n", res)

            const body = (await res.json()) as SyncUserResult;
            return body;
        } catch (err) {
            console.error("[ApiService.assignUserRole]", err);
            console.log(err)
            throw err;
        }
    }

    static async getUserDetails(meta: MetaInfo): Promise<UserDetailsData> {
        try {
            const res = await fetch(`${API_URL}/api/v1/user/status`, {
                method: "GET",
                headers: generateMetaHeaders(meta),
            });

            console.log(res)

            if (res.status === 401) {
                const message = await res.text();
                throw new UnauthorizedError(message || "Unauthorized");
            }

            if(res.status === 404) {
                throw new NotFoundError("User not found");
            }

            if (!res.ok) {
                throw new Error(`Failed to fetch user details: ${res.status} ${res.statusText}`);
            }

            const response: ApiResponse<UserDetailsData> = await res.json() as ApiResponse<UserDetailsData>;

            if (!response.result) {
                throw new Error("No data received in API response");
            }

            return response.result;
        } catch (err) {
            console.error("[ApiService.getUserDetails]", err);
            throw err;
        }
    }

    /**
     * Verifies if the current user has god-mode access
     * Returns true if user is god-mode, false otherwise
     */
    static async verifyGodMode(meta: MetaInfo): Promise<boolean> {
        try {
            const res = await fetch(`${API_URL}/api/v1/admin/verify-god`, {
                method: "GET",
                headers: generateMetaHeaders(meta),
            });

            if (res.status === 401 || res.status === 403) {
                // Not authorized or forbidden = not god mode
                return false;
            }

            if (!res.ok) {
                console.error("[ApiService.verifyGodMode] Unexpected status:", res.status);
                return false;
            }

            const response: ApiResponse<{ is_god: boolean }> = await res.json() as ApiResponse<{ is_god: boolean }>;
            return response.result?.is_god || false;
        } catch (err) {
            console.error("[ApiService.verifyGodMode]", err);
            return false;
        }
    }

    /**
     * Links an existing registered user to a VA with their callsign
     * This is for users who are already registered but not linked to the current VA
     */
    static async linkUserToVA(meta: MetaInfo, callsign: string): Promise<ApiResponse<any>> {
        try {
            const res = await fetch(`${API_URL}/api/v1/user/register/link`, {
                method: "POST",
                headers: {
                    ...generateMetaHeaders(meta),
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

            if (!res.ok) {
                throw new Error(`Failed to link user to VA: ${res.status} ${res.statusText}`);
            }

            const response: ApiResponse<any> = await res.json() as ApiResponse<any>;

            if (!response.result) {
                throw new Error("No data received in API response");
            }

            return response;
        } catch (err) {
            console.error("[ApiService.linkUserToVA]", err);
            throw err;
        }
    }

    static async getPilotStats(meta: MetaInfo): Promise<ApiResponse<PilotStatsData>> {
        try {
            const res = await fetch(`${API_URL}/api/v1/pilot/stats`, {
                method: "GET",
                headers: generateMetaHeaders(meta),
            });

            if (res.status === 401) {
                const message = await res.text();
                throw new UnauthorizedError(message || "Unauthorized");
            }

            if (!res.ok) {
                throw new Error(`Failed to fetch pilot stats: ${res.status} ${res.statusText}`);
            }

            const response: ApiResponse<PilotStatsData> = await res.json() as ApiResponse<PilotStatsData>;

            if (!response.result) {
                throw new Error("No data received in API response");
            }

            return response;
        } catch (err) {
            console.error("[ApiService.getPilotStats]", err);
            throw err;
        }
    }

    /**
     * Fetch PIREP configuration for the current user's flight
     * Returns available flight modes with validation status and field definitions
     */
    static async getPirepConfig(meta: MetaInfo): Promise<PirepConfigResponse> {
        try {
            const res = await fetch(`${API_URL}/api/v1/pireps/config`, {
                method: "GET",
                headers: generateMetaHeaders(meta),
            });

            if (res.status === 401) {
                const message = await res.text();
                throw new UnauthorizedError(message || "Unauthorized");
            }

            if (!res.ok) {
                let errorMessage = `Failed to fetch PIREP config: ${res.status} ${res.statusText}`;
                try {
                    const errorData = await res.json() as any;
                    if (errorData.message) {
                        errorMessage = errorData.message;
                    }
                } catch (parseErr) {
                    // JSON parsing failed, use generic error message
                }
                throw new Error(errorMessage);
            }

            const response: PirepConfigResponse = await res.json() as PirepConfigResponse;

            if (!response.result) {
                throw new Error("No data received in API response");
            }

            return response;
        } catch (err) {
            console.error("[ApiService.getPirepConfig]", err);
            throw err;
        }
    }

    /**
     * Submit a PIREP for filing
     * Handles all flight modes with mode-specific validation
     */
    static async submitPirep(meta: MetaInfo, pirepData: PirepSubmitRequest): Promise<PirepSubmitResponse> {
        try {
            const res = await fetch(`${API_URL}/api/v1/pireps/submit`, {
                method: "POST",
                headers: {
                    ...generateMetaHeaders(meta),
                    "Content-Type": "application/json"
                },
                body: JSON.stringify(pirepData)
            });

            if (res.status === 401) {
                const message = await res.text();
                throw new UnauthorizedError(message || "Unauthorized");
            }

            if (res.status === 403) {
                const body = await res.json() as any;
                const errorMessage = body.error?.message || body.message || "Access denied";
                throw new PermissionDeniedError(errorMessage);
            }

            if (!res.ok) {
                // Try to parse error response
                try {
                    const errorResponse = await res.json() as any;
                    // Extract error from nested result structure if present
                    const result = errorResponse.result || {};
                    // Return error response with proper structure
                    return {
                        status: errorResponse.status || "error",
                        message: errorResponse.message || "PIREP submission failed",
                        result: {
                            success: false,
                            error_message: result.error_message || errorResponse.message || "PIREP submission failed",
                            error: result.error || errorResponse.error,
                        } as any,
                    } as PirepSubmitResponse;
                } catch (parseErr) {
                    // If JSON parsing fails, return generic error
                    return {
                        status: "error",
                        message: `Failed to submit PIREP: ${res.status} ${res.statusText}`,
                        result: {
                            success: false,
                            error_message: `HTTP ${res.status}: ${res.statusText}`,
                        } as any,
                    } as PirepSubmitResponse;
                }
            }

            const response: PirepSubmitResponse = await res.json() as PirepSubmitResponse;
            return response;
        } catch (err) {
            console.error("[ApiService.submitPirep]", err);
            throw err;
        }
    }

    /**
     * Generate a presigned dashboard link for web UI access
     * Returns a single-use URL that expires in 15 minutes
     */
    static async generateDashboardLink(meta: MetaInfo): Promise<ApiResponse<{ url: string; expires_in: number }>> {
        try {
            const res = await fetch(`${API_URL}/api/v1/auth/generate-dashboard-link`, {
                method: "POST",
                headers: {
                    ...generateMetaHeaders(meta),
                    "Content-Type": "application/json"
                },
                body: JSON.stringify({})
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
                throw new Error(`Failed to generate dashboard link: ${res.status} ${res.statusText}`);
            }

            const response: ApiResponse<{ url: string; expires_in: number }> = await res.json() as ApiResponse<{ url: string; expires_in: number }>;

            if (!response.result) {
                throw new Error("No data received in API response");
            }

            return response;
        } catch (err) {
            console.error("[ApiService.generateDashboardLink]", err);
            throw err;
        }
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
            const res = await fetch(`${API_URL}/api/v1/signed-link`, {
                method: "POST",
                headers: {
                    ...generateMetaHeaders(meta),
                    "Content-Type": "application/json"
                },
                body: JSON.stringify({
                    redirect_to: redirectTo,
                    ttl_minutes: ttlMinutes || 15
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

            const response: ApiResponse<{ url: string; expires_in: number; redirect_to: string }> = await res.json() as ApiResponse<{ url: string; expires_in: number; redirect_to: string }>;

            if (!response.result) {
                throw new Error("No data received in API response");
            }

            return response;
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
            const res = await fetch(`${API_URL}/api/v1/memberships/join`, {
                method: "POST",
                headers: {
                    ...generateMetaHeaders(meta),
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

            if (!response.result) {
                throw new Error("No data received in API response");
            }

            return response.result;
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
        try {
            const res = await fetch(`${API_URL}/api/v1/events?active_only=true`, {
                method: "GET",
                headers: generateMetaHeaders(meta),
            });

            if (res.status === 401) {
                const message = await res.text();
                throw new UnauthorizedError(message || "Unauthorized");
            }

            if (res.status === 403) {
                const body = await res.json() as any;
                const errorMessage = body.error?.message || body.message || "Access denied";
                throw new PermissionDeniedError(errorMessage);
            }

            if (!res.ok) {
                // Try to parse error response
                try {
                    const body = await res.json() as any;
                    const errorMessage = body.error?.message || body.message || `Failed to fetch active events: ${res.status} ${res.statusText}`;
                    throw new Error(errorMessage);
                } catch (parseErr) {
                    // If parsing fails, use status text
                    throw new Error(`Failed to fetch active events: ${res.status} ${res.statusText}`);
                }
            }

            const response: EventsResponse = await res.json() as EventsResponse;

            if (!response.result) {
                throw new Error("No data received in API response");
            }

            return response;
        } catch (err) {
            console.error("[ApiService.getActiveEvents]", err);
            throw err;
        }
    }

    /**
     * Get a specific event leg by leg number from an event
     * Returns the leg matching the leg_number
     */
    static async getEventLegByNumber(meta: MetaInfo, eventId: string, legNumber: number): Promise<TourLegResponse> {
        try {
            const res = await fetch(`${API_URL}/api/v1/events/${eventId}/legs`, {
                method: "GET",
                headers: generateMetaHeaders(meta),
            });

            if (res.status === 401) {
                const message = await res.text();
                throw new UnauthorizedError(message || "Unauthorized");
            }

            if (res.status === 403) {
                const body = await res.json() as any;
                const errorMessage = body.error?.message || body.message || "Access denied";
                throw new PermissionDeniedError(errorMessage);
            }

            if (!res.ok) {
                // Try to parse error response
                try {
                    const body = await res.json() as any;
                    const errorMessage = body.error?.message || body.message || `Failed to fetch event legs: ${res.status} ${res.statusText}`;
                    throw new Error(errorMessage);
                } catch (parseErr) {
                    // If parsing fails, use status text
                    throw new Error(`Failed to fetch event legs: ${res.status} ${res.statusText}`);
                }
            }

            const response: ApiResponse<TourLegResponse[]> = await res.json() as ApiResponse<TourLegResponse[]>;

            if (!response.result) {
                throw new Error("No data received in API response");
            }

            const leg = response.result.find(l => l.leg_number === legNumber);
            if (!leg) {
                throw new Error(`Leg number ${legNumber} not found in event`);
            }

            return leg;
        } catch (err) {
            console.error("[ApiService.getEventLegByNumber]", err);
            throw err;
        }
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
        try {
            const res = await fetch(`${API_URL}/api/v1/events/${eventId}/legs/${legId}/additional-data`, {
                method: "PATCH",
                headers: {
                    ...generateMetaHeaders(meta),
                    "Content-Type": "application/json"
                },
                body: JSON.stringify({ additional_data: additionalData })
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
                throw new Error(`Failed to update leg additional data: ${res.status} ${res.statusText}`);
            }

            const response: ApiResponse<TourLegResponse> = await res.json() as ApiResponse<TourLegResponse>;

            if (!response.result) {
                throw new Error("No data received in API response");
            }

            return response.result;
        } catch (err) {
            console.error("[ApiService.updateEventLegAdditionalData]", err);
            throw err;
        }
    }
}




export interface SyncUserPayload {
    user_id: string;
    callsign: string;
}
export interface SyncUserResult {
    status: string;
    message?: string;  // add this
    data?: any;
    error?: string;
}