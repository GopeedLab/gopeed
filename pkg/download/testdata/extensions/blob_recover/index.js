gopeed.events.onResolve(async function (ctx) {
    if (!ctx.req.url.endsWith("/recover")) {
        return;
    }

    const url = gopeed.runtime.blob.createObjectURL(async ({ offset = 0 }) => new ReadableStream({
        async start(controller) {
            if (offset > 0) {
                controller.close();
                return;
            }
            controller.enqueue(new TextEncoder().encode("stale\n"));
            await new Promise((resolve) => setTimeout(resolve, 50));
            controller.error(new Error("expired"));
        },
    }), { size: 10 });

    ctx.res = {
        name: "blob-recover",
        files: [
            {
                name: "recover.txt",
                size: 10,
                req: {
                    url,
                }
            }
        ]
    };
});

gopeed.events.onError(async function (ctx) {
    const req = ctx.task.meta.req;
    if (!req.rawUrl || !req.rawUrl.endsWith("/recover")) {
        return;
    }
    req.labels = req.labels || {};
    if (req.labels.recovered === "true") {
        return;
    }

    req.labels.recovered = "true";
    req.url = gopeed.runtime.blob.createObjectURL(new Blob(["recovered\n"], { type: "text/plain" }));
    ctx.task.continue();
});
