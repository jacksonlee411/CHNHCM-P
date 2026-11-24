import { execFileSync } from "node:child_process";
import { fileURLToPath } from "node:url";

export const localRoot = fileURLToPath(new URL("..", import.meta.url));
function runtime(...args) {
  try {
    return execFileSync("python3", ["scripts/local-bu.py", ...args], {
      cwd: localRoot, encoding: "utf8", stdio: ["ignore", "pipe", "pipe"],
    }).trim();
  } catch (error) {
    throw new Error(error.stderr?.toString().trim() || "Local runtime check failed before browser login");
  }
}

// Contains only ports and checkout identity; never reads runtime credentials.
export function localConnection() {
  return JSON.parse(runtime("connection"));
}

export function verifyLocalTarget(origin = localConnection().webOrigin) {
  runtime("verify", "--origin", origin);
}
