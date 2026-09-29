type LogLevel = "debug" | "info" | "warn" | "error";

type LogFields = Record<string, unknown>;

const levelRank: Record<LogLevel, number> = {
    debug: 10,
    info: 20,
    warn: 30,
    error: 40,
};

const sensitiveKeys = new Set([
    "discord_bot_token",
    "bot_token",
    "api_key",
    "authorization",
    "x-api-key",
    "token",
    "password",
    "secret",
    "body",
    "headers",
    "modal_values",
    "params",
]);

function configuredLevel(): LogLevel {
    const raw = (process.env.LOG_LEVEL || "info").toLowerCase();
    if (raw === "debug" || raw === "info" || raw === "warn" || raw === "error") {
        return raw;
    }
    return "info";
}

function shouldLog(level: LogLevel): boolean {
    return levelRank[level] >= levelRank[configuredLevel()];
}

function normalizeKey(key: string): string {
    return key.toLowerCase();
}

function sanitizeValue(key: string, value: unknown): unknown {
    const normalizedKey = normalizeKey(key);
    if (sensitiveKeys.has(normalizedKey) || sensitiveKeys.has(normalizedKey.replace(/_/g, "-"))) {
        return "[REDACTED]";
    }

    if (value instanceof Error) {
        return {
            error_name: value.name,
            error_message: value.message,
        };
    }

    if (Array.isArray(value)) {
        return value.map((item, index) => sanitizeValue(`${key}_${index}`, item));
    }

    if (value && typeof value === "object") {
        const sanitized: LogFields = {};
        for (const [nestedKey, nestedValue] of Object.entries(value as LogFields)) {
            sanitized[nestedKey] = sanitizeValue(nestedKey, nestedValue);
        }
        return sanitized;
    }

    return value;
}

export function sanitizeLogFields(fields: LogFields): LogFields {
    const sanitized: LogFields = {};
    for (const [key, value] of Object.entries(fields)) {
        sanitized[key] = sanitizeValue(key, value);
    }
    return sanitized;
}

function write(level: LogLevel, event: string, fields: LogFields = {}): void {
    if (!shouldLog(level)) return;

    const line = JSON.stringify({
        timestamp: new Date().toISOString(),
        level,
        service: "comrade-bot",
        env: process.env.APP_ENV || process.env.NODE_ENV || "development",
        event,
        ...sanitizeLogFields(fields),
    });

    if (level === "error") {
        process.stderr.write(`${line}\n`);
        return;
    }
    process.stdout.write(`${line}\n`);
}

export const logger = {
    debug: (event: string, fields?: LogFields) => write("debug", event, fields),
    info: (event: string, fields?: LogFields) => write("info", event, fields),
    warn: (event: string, fields?: LogFields) => write("warn", event, fields),
    error: (event: string, fields?: LogFields) => write("error", event, fields),
};

export function errorFields(error: unknown): LogFields {
    if (error instanceof Error) {
        return {
            error_name: error.name,
            error_message: error.message,
        };
    }

    return {
        error_name: "UnknownError",
        error_message: String(error),
    };
}
