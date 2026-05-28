import { Container } from "@cloudflare/containers";
import { Hono } from "hono";
import { ide } from "./ide";

export class Workspace extends Container<Env> {
	defaultPort = 8080;
	sleepAfter = "10m";
	envVars = {
		WORKSPACE_DIR: "/workspace",
	};

	override onStart() {
		console.log("workspace started");
	}

	override onStop() {
		console.log("workspace stopped");
	}

	override onError(error: unknown) {
		console.log("workspace error:", error);
	}
}

const app = new Hono<{ Bindings: Env }>();

app.get("/", (c) => {
	return c.redirect(`/ws/${randomId()}`);
});

app.get("/ws/:id", (c) => {
	return c.html(ide(c.req.param("id")));
});

// Proxy everything under /ws/:id/* to the matching workspace container.
app.all("/ws/:id/*", async (c) => {
	const id = c.req.param("id");
	const doId = c.env.WORKSPACE.idFromName(`workspace-${id}`);
	const stub = c.env.WORKSPACE.get(doId);

	const url = new URL(c.req.url);
	url.pathname = url.pathname.replace(`/ws/${id}`, "") || "/";

	const upstream = new Request(url.toString(), c.req.raw);
	return stub.fetch(upstream);
});

export default app;

function randomId(): string {
	const bytes = new Uint8Array(6);
	crypto.getRandomValues(bytes);
	return Array.from(bytes, (b) => b.toString(16).padStart(2, "0")).join("");
}
