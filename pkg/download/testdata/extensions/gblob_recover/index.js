gopeed.events.onResolve(async function (ctx) {
    if (!ctx.req.url.endsWith("/recover")) {
        return;
    }

    const url = URL.createObjectURL(new ReadableStream({
        async start(controller) {
            controller.enqueue(new TextEncoder().encode("stale\n"));
            await new Promise((resolve) => setTimeout(resolve, 50));
            controller.error(new Error("expired"));
        },
    }));

    ctx.res = {
        name: "gblob-recover",
        files: [
            {
                name: "recover.txt",
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
    req.url = URL.createObjectURL(new Blob(["recovered\n"], { type: "text/plain" }));
    ctx.task.continue();
});
