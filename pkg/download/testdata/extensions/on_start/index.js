gopeed.events.onStart(async function (ctx) {
    ctx.task.meta.req.url = "https://github.com";
    ctx.task.meta.req.labels['modified'] = 'true';
});

