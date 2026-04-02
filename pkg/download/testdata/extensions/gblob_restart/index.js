function createRangeGBlobUrl(target) {
  return URL.createObjectURL(async (offset) => {
    const headers = {};
    if (offset > 0) {
      headers.Range = `bytes=${offset}-`;
    }
    const response = await fetch(target, { headers });
    if (!response.body) {
      throw new Error('empty response body');
    }
    return response.body;
  });
}

gopeed.events.onResolve(async function (ctx) {
  if (!ctx.req.url.includes('/restart-range')) {
    return;
  }

  ctx.res = {
    name: 'gblob-restart',
    range: true,
    files: [
          {
            name: 'restart.bin',
            size: 262144,
            req: {
              url: createRangeGBlobUrl(ctx.req.rawUrl || ctx.req.url),
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
  if (!req || !req.rawUrl || !req.url || !req.url.startsWith('gblob:')) {
    return;
  }
  req.labels = req.labels || {};
  if (req.labels.rebuilt === 'true') {
    return;
  }

  try {
    req.url = createRangeGBlobUrl(req.rawUrl);
    req.labels.started = 'true';
    req.labels.rebuilt = 'true';
    ctx.task.continue();
  } catch (error) {
    req.labels.rebuildError = String(error);
    throw error;
  }
});
