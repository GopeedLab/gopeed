gopeed.events.onResolve(async function (ctx) {
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

    const transform = new TransformStream();
    const writer = transform.writable.getWriter();
    const url = URL.createObjectURL(transform.writable);

    if (ctx.req.url.endsWith("/stream-unknown")) {
        (async () => {
            await writer.write(new TextEncoder().encode("line 1\n"));
            await new Promise((resolve) => setTimeout(resolve, 120));
            await writer.write(new TextEncoder().encode("line 2\n"));
            await writer.close();
        })();

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

    await writer.write(new TextEncoder().encode("line 1\n"));
    await writer.write(new TextEncoder().encode("line 2\n"));
    await writer.close();

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
