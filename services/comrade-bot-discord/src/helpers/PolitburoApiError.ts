import type { components } from "../generated/politburo-api";

export type PolitburoErrorCode = components["schemas"]["CreateUserErrorCode"] | string;

type ErrorBody = {
    error?: { code?: string; message?: string };
    message?: string;
};

/**
 * Politburo rewrite error envelope: { error: { code, message } }.
 */
export class PolitburoApiError extends Error {
    readonly code: PolitburoErrorCode;
    readonly httpStatus: number;

    constructor(httpStatus: number, code: PolitburoErrorCode, message: string) {
        super(message);
        this.name = "PolitburoApiError";
        this.httpStatus = httpStatus;
        this.code = code;
        Object.setPrototypeOf(this, PolitburoApiError.prototype);
    }

    static fromBody(httpStatus: number, body: unknown): PolitburoApiError {
        const parsed = (body ?? {}) as ErrorBody;
        const code = parsed.error?.code ?? "UNKNOWN";
        const message = parsed.error?.message ?? parsed.message ?? "Request failed";
        return new PolitburoApiError(httpStatus, code, message);
    }

    static isPolitburoApiError(err: unknown): err is PolitburoApiError {
        return err instanceof PolitburoApiError;
    }
}
