import dotenv from "dotenv";
import path from "path";

// Must run before any module reads process.env (imports are hoisted above dotenv in index.ts).
dotenv.config({ path: path.resolve(__dirname, "..", ".env") });
