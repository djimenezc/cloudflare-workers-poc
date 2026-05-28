# Vega Ephemeral Dev Environment

> Personal development environments at the edge.

A simplified Gitpod-style dev environment running on **Cloudflare Workers Containers**. Each workspace is a Durable-Object-backed container that spins up on demand, sleeps when idle, and gives you a browser IDE — file tree, Monaco editor, shell terminal, and a live preview with hot reload.

## Architecture

```
Browser SPA                Worker (Hono)                 Workspace (per-id DO)
┌───────────────┐         ┌────────────────┐            ┌────────────────────┐
│ File tree     │         │ /ws/:id        │            │ Go agent           │
│ Monaco editor │ ◄─────► │ proxies HTTP   │ ◄────────► │  GET /api/tree     │
│ xterm.js term │         │ proxies WS     │            │  GET/PUT /api/file │
│ Preview iframe│         │ serves IDE     │            │  WS  /api/terminal │
└───────────────┘         └────────────────┘            │  WS  /api/events   │
                                                       │  GET /preview/*    │
                                                       └────────────────────┘
                                                       Alpine + bash + pty
```

- **Browser SPA**: single HTML doc served by the Worker. Monaco + xterm via CDN, vanilla JS.
- **Worker**: Hono router. Spawns/wakes the container Durable Object keyed by workspace id and proxies HTTP + WebSockets through.
- **Workspace container**: a Go agent inside an Alpine image. Exposes a file API, a pty-backed terminal over WebSocket, an fsnotify event stream, and a static `/preview/*` route.

## Features

| Feature           | How it works                                                                   |
|-------------------|--------------------------------------------------------------------------------|
| Spawn workspace   | Visit `/ws/<id>` — `WORKSPACE.idFromName("workspace-<id>")` resolves a unique DO. |
| Browser IDE       | Monaco editor loaded from CDN; `Ctrl+S` / `Cmd+S` saves to the container.      |
| File tree         | `GET /api/tree` returns the recursive tree under `/workspace`.                 |
| Terminal          | `WS /api/terminal` opens a pty-backed `bash -l` (via `creack/pty`).            |
| Hot reload        | `fsnotify` events stream over `WS /api/events`; the browser reloads the preview iframe when `public/*` changes. |
| Ephemeral storage | Workspace lives in the container; reset on cold start. Persistence = v2.       |

## Prerequisites

- [Node.js](https://nodejs.org/) 18+
- [Docker](https://www.docker.com/) running locally (for `make dev`)
- Cloudflare account — account ID set in `wrangler.jsonc`
- `npx wrangler login`

## Quick start

```bash
make install
make tidy        # refresh Go module deps
npx wrangler login
make dev         # opens http://localhost:8787 -> redirects to a fresh /ws/<id>
```

Open `http://localhost:8787` in a browser. The Worker redirects you to `/ws/<random-id>` and the IDE boots. The seed workspace lives at `container_src/seed/` and contains `public/index.html` — edit it in Monaco, hit save, watch the preview reload.

## Demo script

1. Visit the root URL — confirm you land on `/ws/<id>`.
2. Click `public/index.html` in the tree, change the `<h1>`, press `Cmd+S`.
3. Preview pane reloads automatically (reload counter ticks up).
4. In the terminal, `ls`, `cat public/index.html`, `echo $WORKSPACE_DIR`, `git init`.
5. Open a second browser tab on `/ws/<a-different-id>` — confirm it's an independent container.
6. Run `npx wrangler containers list` (or `make list`) to see live instances.

## Project layout

```
.
├── Dockerfile                 # Alpine + Go agent
├── Makefile                   # make install / tidy / dev / deploy
├── wrangler.jsonc             # Worker config + container binding
├── package.json
├── tsconfig.json
├── src/
│   ├── index.ts               # Worker entry — routes /ws/:id and proxies API
│   └── ide.ts                 # The browser IDE HTML (inline Monaco + xterm)
└── container_src/
    ├── go.mod
    ├── main.go                # HTTP server, route wiring
    ├── files.go               # file tree + read/write
    ├── terminal.go            # WS pty terminal
    ├── watcher.go             # fsnotify event stream
    └── seed/                  # seeded workspace contents
        ├── README.md
        └── public/index.html
```

## Deployment

```bash
make deploy
```

That runs `wrangler deploy`, which builds the linux/amd64 image, pushes it to Cloudflare's container registry, and rolls out the Worker with the `Workspace` Durable Object binding.

After deploy, visit `https://vega-ephemeral-dev.<your-subdomain>.workers.dev`.

## Limitations (intentional, for the POC)

- **No persistence.** The workspace resets when the container sleeps. Hook the file API up to R2 or DO storage in v2.
- **No auth.** Anyone with the URL can use any workspace id. Add Cloudflare Access or a JWT check in the worker before showing this off externally.
- **One port per container.** The preview is served by the Go agent itself from `/workspace/public`. To proxy a user's own dev server, the agent would need to forward `/preview/*` to `localhost:3000` (or similar) — easy follow-up.
- **No image install API.** All workspaces start from the same Alpine + Go image. Per-language templates would mean either multiple container classes or a richer base image.

## Pitch

> Personal development environments at the edge.
>
> Each developer gets a fresh, sandboxed container, started in milliseconds on Cloudflare's edge, with a browser IDE that ships with the URL. No VPN, no `kubectl`, no laptop setup — open the link and you're coding.
