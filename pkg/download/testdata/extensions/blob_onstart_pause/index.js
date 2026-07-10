gopeed.events.onResolve(async function (ctx) {
  if (!ctx.req.url.endsWith('/onstart-pause')) {
    return;
  }
  const url = gopeed.runtime.blob.createObjectURL(new Blob(['old-source'], { type: 'text/plain' }));
  ctx.res = {
    name: 'blob-onstart-pause',
    files: [
      {
        name: 'onstart-pause.txt',
        size: 10,
        req: { url },
      },
    ],
  };
});

gopeed.events.onStart(async function (ctx) {
  const req = ctx.task.meta.req;
  req.labels = req.labels || {};
  if (req.labels.replaced === 'true') {
    return;
  }
  req.url = gopeed.runtime.blob.createObjectURL(new Blob(['new-source'], { type: 'text/plain' }));
  req.labels.replaced = 'true';
  ctx.task.pause();
});
