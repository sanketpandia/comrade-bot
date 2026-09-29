/** Unwrap Politburo rewrite `{ data }` or legacy `{ result }` payloads. */
export function unwrapApiData<T>(body: Record<string, unknown> | null | undefined): T | undefined {
    if (!body) {
        return undefined;
    }
    if (body.data !== undefined && body.data !== null) {
        return body.data as T;
    }
    if (body.result !== undefined && body.result !== null) {
        return body.result as T;
    }
    return undefined;
}
