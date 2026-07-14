gopeed.events.onStart(async function (ctx) {
    gopeed.logger.info("url", ctx.task.meta.req.url);
    ctx.task.setUrl("https://github.com");
    ctx.task.meta.req.putLabel('modified', 'true');
});
