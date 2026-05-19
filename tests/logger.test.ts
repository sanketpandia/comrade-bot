import test from "node:test";
import assert from "node:assert/strict";
import { errorFields, sanitizeLogFields } from "../src/infra/logger";

test("sanitizeLogFields redacts known sensitive fields recursively", () => {
    const sanitized = sanitizeLogFields({
        DISCORD_BOT_TOKEN: "bot-token",
        API_KEY: "api-key",
        Authorization: "Bearer secret",
        nested: {
            token: "nested-token",
            safe: "visible",
            headers: {
                "X-API-Key": "header-key",
            },
        },
        command: "status",
    });

    assert.equal(sanitized.DISCORD_BOT_TOKEN, "[REDACTED]");
    assert.equal(sanitized.API_KEY, "[REDACTED]");
    assert.equal(sanitized.Authorization, "[REDACTED]");
    assert.deepEqual(sanitized.nested, {
        token: "[REDACTED]",
        safe: "visible",
        headers: "[REDACTED]",
    });
    assert.equal(sanitized.command, "status");
});

test("errorFields returns sanitized error identity without stack traces", () => {
    const fields = errorFields(new TypeError("bad input"));

    assert.deepEqual(fields, {
        error_name: "TypeError",
        error_message: "bad input",
    });
    assert.equal(Object.prototype.hasOwnProperty.call(fields, "stack"), false);
});
