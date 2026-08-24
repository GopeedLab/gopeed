gopeed.events.onStart(async function (ctx) {
    gopeed.logger.info("url", ctx.task.meta.req.url);
    await ctx.task.setUrl("https://github.com");
    await ctx.task.meta.req.putLabel('modified', 'true');
});
