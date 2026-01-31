import fetch from "node-fetch";
import { HealthApiResponse, InitRegistrationResponse, ApiResponse, FlightHistoryPage, InitServerResponse, LiveFlightRecord, UserDetailsData, PilotStatsData, PirepConfigResponse, PirepSubmitResponse, PirepSubmitRequest } from "../types/Responses";
import { MetaInfo } from "../types/DiscordInteraction";
import { generateMetaHeaders } from "../helpers/utils";
import { UnauthorizedError } from "../helpers/UnauthorizedException";
import { PermissionDeniedError } from "../helpers/PermissionDeniedException";
import { NotFoundError } from "../helpers/NotFoundException";

const API_URL = process.env.API_URL ?? "http://localhost:8080";

type InitServerResult = ApiResponse<InitServerResponse>;

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
        lastFlight: string,
        callsign?: string
    ): Promise<InitRegistrationResponse> {
        try {
            const payload: any = {
                ifc_id: ifcId,
                last_flight: lastFlight
            };

            // Add callsign if provided (for VA servers)
            if (callsign) {
                payload.callsign = callsign;
            }

            const res = await fetch(`${API_URL}/api/v1/user/register/init`, {
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

            if (!res.ok) {
                throw new Error(`Failed to fetch initRegistration: ${res.status} ${res.statusText}`);
            }
            const response: ApiResponse<InitRegistrationResponse> = await res.json() as ApiResponse<InitRegistrationResponse>;

            if (!response.data) {
                throw new Error("No data received in API response");
            }
            return response.data;
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
        callsignSuffix: string,
        iconURL?: string
    ): Promise<InitServerResult> {
        try {
            const res = await fetch(`${API_URL}/api/v1/server/init`, {
                method: "POST",
                headers: generateMetaHeaders(meta),
                body: JSON.stringify({
                    va_code: code,
                    name,
                    callsign_prefix: callsignPrefix,
                    callsign_suffix: callsignSuffix,
                    ...(iconURL && { icon_url: iconURL })
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

            // Always try to read the JSON body (success or error)
            const body: InitServerResult = await res.json() as InitServerResult;

            /*          ─── decide at the CALL-SITE ───
               - If res.ok === true  → body.status should be true.
               - If res.ok === false → body.status is false and body.data.steps
                                       tells which stage failed.                */
            return body;
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

            if (!response.data) {
                throw new Error("No data received in API response");
            }

            // Include the response_time from the API response
            return {
                ...response.data,
                response_time: response.response_time
            };


        } catch (err) {
            console.error("[ApiService.getLogbook]", err);
            throw err
        }
    }

    static async getLiveFlights(meta: MetaInfo): Promise<LiveFlightRecord[]> {
        try {
            const res = await fetch(`${API_URL}/api/v1/va/live`, {
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

            const response: ApiResponse<LiveFlightRecord[]> = await res.json() as ApiResponse<LiveFlightRecord[]>;

            if (!response.data) {
                throw new Error("No data received in API response");
            }

            return response.data;

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

            if (!response.data) {
                throw new Error("No data received in API response");
            }

            return response.data;
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
            return response.data?.is_god || false;
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

            if (!response.data) {
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

            if (!response.data) {
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

            if (!response.data) {
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

            if (!res.ok) {
                const errorResponse = await res.json() as PirepSubmitResponse;
                return errorResponse;
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

            if (!response.data) {
                throw new Error("No data received in API response");
            }

            return response;
        } catch (err) {
            console.error("[ApiService.generateDashboardLink]", err);
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