gopeed.events.onDone(async function (ctx) {
    gopeed.logger.info("url", ctx.task.meta.req.url);
    ctx.task.meta.req.labels['modified'] = 'true';
});

