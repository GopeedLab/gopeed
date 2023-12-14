gopeed.events.onResolve(async function (ctx) {
    const aaa = {};
    // access undefined property
    gopeed.logger.info(aaa.bbb.ccc);

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
