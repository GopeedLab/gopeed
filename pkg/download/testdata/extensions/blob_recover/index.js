gopeed.events.onResolve(async function (ctx) {
    const isRange = ctx.req.url.endsWith("/recover-range");
    if (!isRange && !ctx.req.url.endsWith("/recover")) {
        return;
    }

    const url = gopeed.runtime.blob.createObjectURL(async ({ offset = 0 }) => new ReadableStream({
        async start(controller) {
            if (offset > 0) {
                controller.close();
                return;
            }
            controller.enqueue(new TextEncoder().encode("stale-prefix-that-must-be-removed\n"));
            await new Promise((resolve) => setTimeout(resolve, 50));
            controller.error(new Error("expired"));
        },
    }), { size: 64, range: isRange });

    ctx.res = {
        name: "blob-recover",
        files: [
            {
                name: "recover.txt",
                size: 64,
                req: {
                    url,
                }
            }
        ]
    };
});

gopeed.events.onError(async function (ctx) {
    const req = ctx.task.meta.req;
    if (!req.rawUrl || (!req.rawUrl.endsWith("/recover") && !req.rawUrl.endsWith("/recover-range"))) {
        return;
    }
    req.labels = req.labels || {};
    if (req.labels.recovered === "true") {
        return;
    }

    req.labels.recovered = "true";
    req.url = gopeed.runtime.blob.createObjectURL(new Blob(["ok\n"], { type: "text/plain" }));
    ctx.task.continue();
});
