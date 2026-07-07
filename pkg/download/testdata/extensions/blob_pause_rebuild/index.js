const payloadSize = 262144;
const chunkSize = 4096;
const payloadText = 'x'.repeat(payloadSize);

function delayedPayloadStream(offset = 0, end = -1) {
  const limit = end >= 0 ? Math.min(end + 1, payloadSize) : payloadSize;
  let pos = offset;
  return new ReadableStream({
    async pull(controller) {
      if (pos >= limit) {
        controller.close();
        return;
      }
      const size = Math.min(chunkSize, limit - pos);
      const chunk = new Uint8Array(size);
      chunk.fill(120);
      pos += size;
      controller.enqueue(chunk);
      await new Promise((resolve) => setTimeout(resolve, 20));
    },
  });
}

function fastPayloadStream(offset = 0, end = -1) {
  const limit = end >= 0 ? Math.min(end + 1, payloadSize) : payloadSize;
  let pos = offset;
  return new ReadableStream({
    pull(controller) {
      if (pos >= limit) {
        controller.close();
        return;
      }
      const size = Math.min(chunkSize, limit - pos);
      const chunk = new Uint8Array(size);
      chunk.fill(120);
      pos += size;
      controller.enqueue(chunk);
    },
  });
}

function createBlobSourceURL(delayed) {
  const blob = new Blob([payloadText], { type: 'application/octet-stream' });
  const originalSlice = blob.slice.bind(blob);
  blob.slice = function (offset = 0, end) {
    if (!delayed) {
      return originalSlice(offset, end);
    }
    const sliceEnd = Number.isFinite(Number(end)) ? Number(end) - 1 : -1;
    return {
      stream() {
        return delayedPayloadStream(Number(offset) || 0, sliceEnd);
      },
    };
  };
  return gopeed.runtime.blob.createObjectURL(blob);
}

function createOpenerSourceURL(delayed) {
  const open = async () => delayed ? delayedPayloadStream(0, -1) : fastPayloadStream(0, -1);
  return gopeed.runtime.blob.createObjectURL(open, {
    size: payloadSize,
    contentType: 'application/octet-stream',
  });
}

function createRangeSourceURL(delayed) {
  const open = async ({ offset = 0, end = -1 }) => delayed ? delayedPayloadStream(offset, end) : fastPayloadStream(offset, end);
  return gopeed.runtime.blob.createObjectURL(open, {
    size: payloadSize,
    range: true,
    contentType: 'application/octet-stream',
  });
}

function modeFromURL(url) {
  if (url.includes('/pause-rebuild-blob')) {
    return 'blob';
  }
  if (url.includes('/pause-rebuild-opener')) {
    return 'opener';
  }
  if (url.includes('/pause-rebuild-range')) {
    return 'range';
  }
  return '';
}

function createPayloadURL(mode, delayed) {
  switch (mode) {
    case 'blob':
      return createBlobSourceURL(delayed);
    case 'opener':
      return createOpenerSourceURL(delayed);
    case 'range':
      return createRangeSourceURL(delayed);
    default:
      throw new Error('unsupported pause rebuild mode: ' + mode);
  }
}

gopeed.events.onResolve(async function (ctx) {
  const mode = modeFromURL(ctx.req.url);
  if (!mode) {
    return;
  }

  ctx.res = {
    name: 'blob-pause-rebuild-' + mode,
    range: mode !== 'opener',
    files: [
      {
        name: 'pause-rebuild-' + mode + '.bin',
        size: payloadSize,
        req: {
          url: createPayloadURL(mode, true),
          rawUrl: ctx.req.rawUrl || ctx.req.url,
          labels: {
            mode: 'pause-rebuild',
            source: mode,
          },
        },
      },
    ],
  };
});

gopeed.events.onError(async function (ctx) {
  const req = ctx.task?.meta?.req;
  const source = req?.labels?.source || modeFromURL(req?.rawUrl || '');
  if (!req || req.labels?.mode !== 'pause-rebuild' || req.labels?.rebuilt === 'true' || !source) {
    return;
  }

  req.url = createPayloadURL(source, false);
  req.labels = req.labels || {};
  req.labels.started = 'true';
  req.labels.rebuilt = 'true';
  ctx.task.continue();
});
