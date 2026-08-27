async function createRangeBlobUrl(target) {
  return await gopeed.runtime.blob.createObjectURL(async ({ offset = 0, end = -1 }) => {
    const headers = {};
    if (offset > 0 || end >= 0) {
      headers.Range = `bytes=${offset}-${end >= 0 ? end : ''}`;
    }
    const response = await fetch(target, { headers });
    if (!response.body) {
      throw new Error('empty response body');
    }
    return response.body;
  }, { size: 262144, range: true });
}

gopeed.events.onResolve(async function (ctx) {
  if (!ctx.req.url.includes('/restart-range')) {
    return;
  }

  ctx.res = {
    name: 'blob-restart',
    range: true,
    files: [
          {
            name: 'restart.bin',
            size: 262144,
            req: {
              url: await createRangeBlobUrl(ctx.req.rawUrl || ctx.req.url),
              rawUrl: ctx.req.rawUrl || ctx.req.url,
              labels: {
                mode: 'restart',
          },
        },
      },
    ],
  };
});

gopeed.events.onError(async function (ctx) {
  const req = ctx.task?.meta?.req;
  if (!req || !req.rawUrl || !req.url || !req.url.includes('/__blob/')) {
    return;
  }
  if (req.labels.rebuilt === 'true') {
    return;
  }

  try {
    await ctx.task.setUrl(await createRangeBlobUrl(req.rawUrl));
    await req.putLabel('started', 'true');
    await req.putLabel('rebuilt', 'true');
    await ctx.task.continue();
  } catch (error) {
    await req.putLabel('rebuildError', String(error));
    throw error;
  }
});
