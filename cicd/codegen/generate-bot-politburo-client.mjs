#!/usr/bin/env node
/**
 * Copies the Politburo openapi-fetch client template into the bot generated tree.
 * Run after openapi-typescript (politburo-api.ts).
 */
import { copyFileSync, mkdirSync } from "node:fs";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";

const __dirname = dirname(fileURLToPath(import.meta.url));
const repoRoot = join(__dirname, "..", "..");
const template = join(__dirname, "bot-politburo-client.ts");
const outDir = join(repoRoot, "services", "comrade-bot-discord", "src", "generated");
const outFile = join(outDir, "politburoClient.ts");

mkdirSync(outDir, { recursive: true });
copyFileSync(template, outFile);
console.log(`Copied politburo client → ${outFile}`);
