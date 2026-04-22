gopeed.events.onResolve(async function (ctx) {
    const encoder = new TextEncoder();

    if (ctx.req.url.endsWith("/blob")) {
        const blob = new Blob(["hello world"], {type: "text/plain"});
        const url = URL.createObjectURL(blob);
        ctx.res = {
            name: "gblob-blob",
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

    if (ctx.req.url.includes("/http-stream")) {
        const source = new URL(ctx.req.url);
        const target = source.searchParams.get("target");
        const pair = source.searchParams.get("pair") === "1";
        const resumable = ctx.req.url.includes("/http-stream-range");
        const fileName = source.searchParams.get("name") || "http-stream.bin";
        const headResponse = await fetch(target, {method: "HEAD"});
        const headLength = parseInt(headResponse.headers.get("content-length") || "", 10);

        const createOpen = () => async (offset) => {
            const headers = {};
            if (offset > 0) {
                headers.Range = `bytes=${offset}-`;
            }
            const response = await fetch(target, {headers});
            if (!response.body) {
                throw new Error("empty response body");
            }
            return response.body;
        };

        ctx.res = {
            name: "gblob-http-stream",
            range: resumable,
            files: pair ? [
                {
                    name: "pair-video.bin",
                    size: Number.isFinite(headLength) ? headLength : 0,
                    req: {
                        url: URL.createObjectURL(createOpen()),
                    }
                },
                {
                    name: "pair-audio.bin",
                    size: Number.isFinite(headLength) ? headLength : 0,
                    req: {
                        url: URL.createObjectURL(createOpen()),
                    }
                }
            ] : [
                {
                    name: fileName,
                    size: Number.isFinite(headLength) ? headLength : 0,
                    req: {
                        url: URL.createObjectURL(createOpen()),
                    }
                }
            ]
        };
        return;
    }

    if (ctx.req.url.endsWith("/stream-range")) {
        const payload = encoder.encode("line 1\nline 2\nline 3\n");
        const firstChunkLength = encoder.encode("line 1\n").byteLength;
        let firstAttempt = true;

        const open = async (offset) => new ReadableStream({
            async start(controller) {
                try {
                    if (offset === 0 && firstAttempt) {
                        firstAttempt = false;
                        controller.enqueue(payload.slice(0, firstChunkLength));
                        await new Promise((resolve) => setTimeout(resolve, 120));
                        controller.error(new Error("resume required"));
                        return;
                    }
                    controller.enqueue(payload.slice(offset));
                    controller.close();
                } catch (err) {
                    controller.error(err);
                }
            }
        });

        const url = URL.createObjectURL(open);

        ctx.res = {
            name: "gblob-stream-range",
            range: true,
            files: [
                {
                    name: "stream-range.txt",
                    size: payload.byteLength,
                    req: {
                        url,
                    }
                }
            ]
        };
        return;
    }

    if (ctx.req.url.endsWith("/stream-unknown")) {
        const url = URL.createObjectURL(new ReadableStream({
            async start(controller) {
                controller.enqueue(encoder.encode("line 1\n"));
                await new Promise((resolve) => setTimeout(resolve, 350));
                controller.enqueue(encoder.encode("line 2\n"));
                controller.close();
            },
        }));

        ctx.res = {
            name: "gblob-stream-unknown",
            files: [
                {
                    name: "stream-unknown.txt",
                    req: {
                        url,
                    }
                }
            ]
        };
        return;
    }

    const url = URL.createObjectURL(new ReadableStream({
        start(controller) {
            controller.enqueue(encoder.encode("line 1\n"));
            controller.enqueue(encoder.encode("line 2\n"));
            controller.close();
        },
    }));

    ctx.res = {
        name: "gblob-stream",
        files: [
            {
                name: "stream.txt",
                size: 14,
                req: {
                    url,
                }
            }
        ]
    };
});
