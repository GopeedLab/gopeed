gopeed.events.onResolve(async function (ctx) {
    for (const method of ['setUrl', 'setLabels', 'putLabel', 'delLabel', 'setMethod', 'putHeader', 'setTrackers']) {
        if (typeof ctx.req[method] !== 'undefined') throw new Error(`onResolve exposes ${method}`);
    }
    ctx.req.labels = { replaced: "true", removed: "true" };
    ctx.req.labels.modified = "true";
    delete ctx.req.labels.removed;
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
