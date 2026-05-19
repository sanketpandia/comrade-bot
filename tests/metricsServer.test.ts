import test from "node:test";
import assert from "node:assert/strict";
import http from "node:http";
import { MetricsServer } from "../src/infra/metricsServer";

function get(path: string, port: number): Promise<{ statusCode: number; body: string }> {
    return new Promise((resolve, reject) => {
        http.get({ hostname: "127.0.0.1", port, path }, response => {
            let body = "";
            response.setEncoding("utf8");
            response.on("data", chunk => {
                body += chunk;
            });
            response.on("end", () => {
                resolve({ statusCode: response.statusCode || 0, body });
            });
        }).on("error", reject);
    });
}

test("metrics server exposes healthz and Prometheus metrics", async () => {
    const port = 19191;
    const server = new MetricsServer({ enabled: true, host: "127.0.0.1", port });

    await server.start();
    try {
        const health = await get("/healthz", port);
        assert.equal(health.statusCode, 200);
        assert.equal(health.body, "ok\n");

        const metrics = await get("/metrics", port);
        assert.equal(metrics.statusCode, 200);
        assert.match(metrics.body, /# HELP comrade_bot_/);
    } finally {
        await server.stop();
    }
});
