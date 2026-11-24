import { verifyLocalTarget } from "../../scripts/local-runtime.mjs";

export default function runtimeSetup() {
  // A healthy foreign service must be rejected before any test sends credentials.
  verifyLocalTarget();
}
