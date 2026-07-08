function createRangeBlobUrl(target) {
  return gopeed.runtime.blob.createObjectURL(async ({ offset = 0, end = -1 }) => {
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
              url: createRangeBlobUrl(ctx.req.rawUrl || ctx.req.url),
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
  req.labels = req.labels || {};
  if (req.labels.rebuilt === 'true') {
    return;
  }

  try {
    req.url = createRangeBlobUrl(req.rawUrl);
    req.labels.started = 'true';
    req.labels.rebuilt = 'true';
    ctx.task.continue();
  } catch (error) {
    req.labels.rebuildError = String(error);
    throw error;
  }
});
