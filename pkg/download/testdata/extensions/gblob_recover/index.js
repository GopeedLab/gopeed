gopeed.events.onResolve(async function (ctx) {
    if (!ctx.req.url.endsWith("/recover")) {
        return;
    }

    const transform = new TransformStream();
    const writer = transform.writable.getWriter();
    const url = URL.createObjectURL(transform.writable);

    (async () => {
        await writer.write(new TextEncoder().encode("stale\n"));
        await new Promise((resolve) => setTimeout(resolve, 50));
        await writer.abort("expired");
    })();

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
