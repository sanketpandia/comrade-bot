/**
 * Thrown when the bot calls a legacy Politburo route that is not in openapi/politburo.yaml
 * and is not mounted on the rewrite server.
 */
export class ApiNotImplementedError extends Error {
    readonly operation: string;

    constructor(operation: string) {
        super(
            `The Politburo API rewrite does not implement "${operation}" yet. This command will be restored when the endpoint is added to the API contract.`,
        );
        this.name = "ApiNotImplementedError";
        this.operation = operation;
    }
}
