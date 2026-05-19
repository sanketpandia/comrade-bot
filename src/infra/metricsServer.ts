import http, { Server } from "http";
import { metrics } from "./metrics";
import { logger } from "./logger";

export interface MetricsServerConfig {
    enabled: boolean;
    host: string;
    port: number;
}

export class MetricsServer {
    private server?: Server;

    constructor(private readonly config: MetricsServerConfig) {}

    async start(): Promise<void> {
        if (!this.config.enabled) {
            logger.info("metrics_server_disabled");
            return;
        }

        this.server = http.createServer(async (req, res) => {
            if (req.url === "/healthz") {
                res.writeHead(200, { "Content-Type": "text/plain" });
                res.end("ok\n");
                return;
            }

            if (req.url !== "/metrics") {
                res.writeHead(404, { "Content-Type": "text/plain" });
                res.end("not found\n");
                return;
            }

            try {
                res.writeHead(200, { "Content-Type": metrics.register.contentType });
                res.end(await metrics.register.metrics());
            } catch (error) {
                logger.error("metrics_render_failed", { error });
                res.writeHead(500, { "Content-Type": "text/plain" });
                res.end("metrics render failed\n");
            }
        });

        await new Promise<void>((resolve, reject) => {
            this.server?.once("error", reject);
            this.server?.listen(this.config.port, this.config.host, () => resolve());
        });

        logger.info("metrics_server_started", {
            host: this.config.host,
            port: this.config.port,
        });
    }

    async stop(): Promise<void> {
        if (!this.server) return;

        await new Promise<void>((resolve, reject) => {
            this.server?.close(error => {
                if (error) {
                    reject(error);
                    return;
                }
                resolve();
            });
        });

        logger.info("metrics_server_stopped");
    }
}
