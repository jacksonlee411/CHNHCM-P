import { defineConfig, type Plugin, type PreviewServer, type ViteDevServer } from "vite";
import { resolve } from "node:path";
import { localConnection, localRoot } from "../../scripts/local-runtime.mjs";

function runtimeIdentity(connection: ReturnType<typeof localConnection>): Plugin {
  const install = (server: ViteDevServer | PreviewServer) => {
    server.middlewares.use("/__chnhcm_local_runtime", (_request, response) => {
      response.setHeader("Content-Type", "application/json");
      response.setHeader("Cache-Control", "no-store");
      const { key, webOrigin, apiOrigin } = connection;
      response.end(JSON.stringify({ key, webOrigin, apiOrigin }));
    });
  };
  return {
    name: "chnhcm-local-runtime-identity",
    configResolved(config) {
      if (config.command !== "serve") return;
      const options = config.isPreview ? config.preview : config.server;
      if (options.host !== "127.0.0.1" || options.port !== connection.ports.web || !options.strictPort) {
        throw new Error("Vite must use this checkout's saved loopback port; use local-bu.py configure");
      }
    },
    configureServer: install,
    configurePreviewServer: install,
  };
}
export default defineConfig(({ command }) => {
  // A frontend build needs source inputs, not a configured local API/DB profile.
  if (command === "build") return {};
  const connection = localConnection();
  const target = connection.apiOrigin;
  // localConnection validates the profile before it enters this path. Each
  // temporary profile owns and removes its optimizer cache with its binaries.
  const profile = connection.profile;
  return {
    cacheDir: resolve(localRoot, ".local", "bin", ...(profile === "default" ? [] : [profile]), "vite"),
    // The CI checkout is immutable for this temporary profile's lifetime.
    optimizeDeps: { include: profile === "ci-browser" && process.env.CI_PLAN ? ["@mui/x-data-grid"] : [] },
    plugins: [runtimeIdentity(connection)],
    server: {
      host: "127.0.0.1", port: connection.ports.web, strictPort: true,
      proxy: { "/api": { target }, "/readyz": { target } },
    },
    preview: {
      host: "127.0.0.1", port: connection.ports.web, strictPort: true,
      proxy: { "/api": { target }, "/readyz": { target } },
    },
  };
});
