gopeed.events.onError(async function (ctx) {
    gopeed.logger.info("url", ctx.task.meta.req.url);
    gopeed.logger.info("error", ctx.error);
    await ctx.task.setUrl("https://github.com");
    await ctx.task.continue();
});
