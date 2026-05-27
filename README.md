# Cloudflare Workers Containers PoC

Proof-of-concept for deploying container apps on [Cloudflare Workers Containers](https://developers.cloudflare.com/containers/).

A **Cloudflare Worker** (TypeScript/Hono) acts as the entry point and routes requests to **container instances** backed by a Go HTTP server. Containers run on-demand as Durable Objects, start automatically on first request, and sleep after inactivity.

## Architecture

```
Internet → Cloudflare Worker (src/index.ts)
                 ↓
          MyContainer (Durable Object)
                 ↓
          Go HTTP server (container_src/main.go)
          running in a linux/amd64 container
```

## Prerequisites

- [Node.js](https://nodejs.org/) 18+
- [Docker](https://www.docker.com/) (running locally for `make dev`)
- Cloudflare account — account ID is already set in `wrangler.jsonc`
- Authenticated wrangler session: `npx wrangler login`

## Quick start

```bash
# Install dependencies
make install

# Authenticate with Cloudflare (one-time)
npx wrangler login

# Run locally (requires Docker)
make dev

# Deploy to Cloudflare
make deploy
```

## Makefile targets

| Target        | Description                                                |
|---------------|------------------------------------------------------------|
| `make install` | Install npm dependencies                                  |
| `make dev`    | Run locally with wrangler dev (requires Docker)            |
| `make deploy` | Build image, push to Cloudflare Registry, deploy worker   |
| `make logs`   | Tail live worker logs                                      |
| `make list`   | List running container instances                           |
| `make images` | List container images in Cloudflare Registry               |
| `make clean`  | Remove `node_modules` and `.wrangler` cache                |

## Project structure

```
.
├── Dockerfile              # Container image (Go server, linux/amd64)
├── Makefile                # Build and deploy automation
├── wrangler.jsonc          # Wrangler config — worker + container bindings
├── package.json            # Node dependencies (wrangler, hono, @cloudflare/containers)
├── tsconfig.json           # TypeScript config
├── worker-configuration.d.ts
├── src/
│   └── index.ts            # Worker entry point — routes requests to containers
└── container_src/
    ├── go.mod
    └── main.go             # Go HTTP server running inside the container
```

## How it works

### Worker (`src/index.ts`)

The Worker extends `Container` from `@cloudflare/containers` to define lifecycle hooks and configuration, then uses Hono to route incoming requests to container instances:

| Route             | Behaviour                                              |
|-------------------|--------------------------------------------------------|
| `GET /`           | List available endpoints                               |
| `GET /container/:id` | Route to a named container (one instance per `:id`) |
| `GET /lb`         | Load-balance across 3 container instances             |
| `GET /singleton`  | Route to a single shared container instance           |
| `GET /error`      | Trigger a container panic (error handling demo)       |

### Container (`container_src/main.go`)

Minimal Go HTTP server that responds with its `MESSAGE` env var and Cloudflare-injected `CLOUDFLARE_DURABLE_OBJECT_ID`. Listens on port `8080`.

### Container config (`wrangler.jsonc`)

```jsonc
"containers": [{ "class_name": "MyContainer", "image": "./Dockerfile", "max_instances": 10 }]
```

`wrangler deploy` builds the Docker image, pushes it to Cloudflare's container registry, and deploys the Worker — no separate `docker push` step needed.

## Deployment

`make deploy` runs `wrangler deploy` which:

1. Builds the Docker image for `linux/amd64`
2. Pushes the image to Cloudflare's internal registry
3. Deploys the Worker with the container binding

After deployment, view the worker at:
`https://vega-containers-poc.<your-subdomain>.workers.dev`

## Useful commands

```bash
# Check container instances
npx wrangler containers list

# Check pushed images
npx wrangler containers images list

# Tail live logs
npx wrangler tail vega-containers-poc

# Regenerate TypeScript env types after changing wrangler.jsonc
npx wrangler types
```
