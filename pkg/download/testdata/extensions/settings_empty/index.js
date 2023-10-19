gopeed.events.onResolve(async function (ctx) {
    if (Object.keys(gopeed.settings).length > 0){
        throw new Error("settings is not empty");
    }

    ctx.res = {
        name: "test",
        files: Array(2).fill(true).map((_, i) => ({
                name: `test-${i}.txt`,
                size: 1024,
                req: {
                    url: ctx.req.url + "/" + i,
                }
            }),
        ),
    };
});
