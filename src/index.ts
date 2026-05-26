import { Container, getContainer, getRandom } from "@cloudflare/containers";
import { Hono } from "hono";

export class MyContainer extends Container<Env> {
	defaultPort = 8080;
	sleepAfter = "2m";
	envVars = {
		MESSAGE: "Hello from Vega!",
	};

	override onStart() {
		console.log("Container started");
	}

	override onStop() {
		console.log("Container stopped");
	}

	override onError(error: unknown) {
		console.log("Container error:", error);
	}
}

const app = new Hono<{ Bindings: Env }>();

app.get("/", (c) => {
	return c.text(
		"Vega Containers PoC\n\n" +
			"GET /container/:id  — route to a named container instance\n" +
			"GET /lb             — load-balance across 3 container instances\n" +
			"GET /singleton      — route to a single shared container instance\n" +
			"GET /error          — trigger a container panic (error handling demo)\n",
	);
});

app.get("/container/:id", async (c) => {
	const id = c.req.param("id");
	const containerId = c.env.MY_CONTAINER.idFromName(`/container/${id}`);
	const container = c.env.MY_CONTAINER.get(containerId);
	return await container.fetch(c.req.raw);
});

app.get("/lb", async (c) => {
	const container = await getRandom(c.env.MY_CONTAINER, 3);
	return await container.fetch(c.req.raw);
});

app.get("/singleton", async (c) => {
	const container = getContainer(c.env.MY_CONTAINER);
	return await container.fetch(c.req.raw);
});

app.get("/error", async (c) => {
	const container = getContainer(c.env.MY_CONTAINER, "error-test");
	return await container.fetch(c.req.raw);
});

export default app;
