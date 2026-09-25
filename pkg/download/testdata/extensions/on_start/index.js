gopeed.events.onStart(async function (ctx) {
    gopeed.logger.info("url", ctx.task.meta.req.url);
    await ctx.task.meta.req.setUrl("https://github.com");
    await ctx.task.meta.req.setHeaders({ "X-Gopeed-Test": "on-start" });
    await ctx.task.meta.req.putLabel('modified', 'true');
});
