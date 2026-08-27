gopeed.events.onResolve(async function (ctx) {
    await ctx.req.setLabels({ replaced: "true", removed: "true" });
    await ctx.req.putLabel("modified", "true");
    await ctx.req.delLabel("removed");
    ctx.res = {
        name: "test",
        files: Array(2).fill(true).map((_, i) => ({
                name: `test-${i}.txt`,
                size: 1024,
                req: {
                    url: ctx.req.url + "/" + i,
                    labels:{
                        "from": gopeed.info.name,
                    }
                }
            }),
        ),
    };
});
