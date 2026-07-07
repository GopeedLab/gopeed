gopeed.events.onResolve(async function (ctx) {
    const encoder = new TextEncoder();

    if (ctx.req.url.endsWith("/blob")) {
        const blob = new Blob(["hello world"], {type: "text/plain"});
        const url = gopeed.runtime.blob.createObjectURL(blob);
        ctx.res = {
            name: "blob-blob",
            files: [
                {
                    name: "hello.txt",
                    size: 11,
                    req: {
                        url,
                    }
                }
            ]
        };
        return;
    }

    if (ctx.req.url.endsWith("/opener-range")) {
        const payload = encoder.encode("line 1\nline 2\nline 3\n");
        const firstChunkLength = encoder.encode("line 1\n").byteLength;
        let firstAttempt = true;

        const open = async ({ offset = 0, end = -1 }) => new ReadableStream({
            async start(controller) {
                try {
                    if (offset === 0 && firstAttempt) {
                        firstAttempt = false;
                        controller.enqueue(payload.slice(0, firstChunkLength));
                        await new Promise((resolve) => setTimeout(resolve, 120));
                        controller.error(new Error("resume required"));
                        return;
                    }
                    controller.enqueue(payload.slice(offset, end >= 0 ? end + 1 : undefined));
                    controller.close();
                } catch (err) {
                    controller.error(err);
                }
            }
        });

        const url = gopeed.runtime.blob.createObjectURL(open, { size: payload.byteLength, range: true });

        ctx.res = {
            name: "blob-opener-range",
            range: true,
            files: [
                {
                    name: "opener-range.txt",
                    size: payload.byteLength,
                    req: {
                        url,
                    }
                }
            ]
        };
        return;
    }

    if (ctx.req.url.endsWith("/opener-unknown")) {
        const url = gopeed.runtime.blob.createObjectURL(async ({ offset = 0 }) => new ReadableStream({
            async start(controller) {
                if (offset > 0) {
                    controller.close();
                    return;
                }
                controller.enqueue(encoder.encode("line 1\n"));
                await new Promise((resolve) => setTimeout(resolve, 1500));
                controller.enqueue(encoder.encode("line 2\n"));
                controller.close();
            },
        }));

        ctx.res = {
            name: "blob-opener-unknown",
            files: [
                {
                    name: "opener-unknown.txt",
                    req: {
                        url,
                    }
                }
            ]
        };
        return;
    }

    const url = gopeed.runtime.blob.createObjectURL(async ({ offset = 0, end = -1 }) => new ReadableStream({
        start(controller) {
            const payload = encoder.encode("line 1\nline 2\n");
            controller.enqueue(payload.slice(offset, end >= 0 ? end + 1 : undefined));
            controller.close();
        },
    }), { size: 14 });

    ctx.res = {
        name: "blob-opener",
        files: [
            {
                name: "opener.txt",
                size: 14,
                req: {
                    url,
                }
            }
        ]
    };
});
