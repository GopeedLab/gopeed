const encoder = new TextEncoder();
const decoder = new TextDecoder();

function byteLength(value) {
  return encoder.encode(value).byteLength;
}

function sourceURL(rawURL) {
  const url = new URL(rawURL);
  url.searchParams.set("source", "1");
  return url.toString();
}

function createFetchDrainedStream(url) {
  return new ReadableStream({
    async start(controller) {
      try {
        const response = await fetch(url);
        const reader = response.body.getReader();
        while (true) {
          const { done, value } = await reader.read();
          if (done) {
            controller.close();
            return;
          }
          controller.enqueue(value);
        }
      } catch (error) {
        controller.error(error);
      }
    },
  });
}

function createFetchDrainedRangeStream(url, offset, end) {
  return new ReadableStream({
    async start(controller) {
      try {
        const headers = {};
        if (offset > 0 || end >= 0) {
          headers.Range = `bytes=${offset}-${end >= 0 ? end : ""}`;
        }
        const response = await fetch(url, { headers });
        const reader = response.body.getReader();
        while (true) {
          const { done, value } = await reader.read();
          if (done) {
            controller.close();
            return;
          }
          controller.enqueue(value);
        }
      } catch (error) {
        controller.error(error);
      }
    },
  });
}

function createMultiplexStreams(url) {
  let videoController;
  let audioController;
  let started = false;

  async function pump() {
    if (started || !videoController || !audioController) {
      return;
    }
    started = true;

    try {
      const response = await fetch(url);
      const reader = response.body.getReader();
      let pending = "";

      while (true) {
        const { done, value } = await reader.read();
        if (done) {
          break;
        }
        pending += decoder.decode(value, { stream: true });

        let newlineIndex;
        while ((newlineIndex = pending.indexOf("\n")) >= 0) {
          const line = pending.slice(0, newlineIndex);
          pending = pending.slice(newlineIndex + 1);
          if (line.startsWith("v:")) {
            videoController.enqueue(encoder.encode(line.slice(2)));
          } else if (line.startsWith("a:")) {
            audioController.enqueue(encoder.encode(line.slice(2)));
          }
        }
      }

      if (pending.startsWith("v:")) {
        videoController.enqueue(encoder.encode(pending.slice(2)));
      } else if (pending.startsWith("a:")) {
        audioController.enqueue(encoder.encode(pending.slice(2)));
      }
      videoController.close();
      audioController.close();
    } catch (error) {
      videoController.error(error);
      audioController.error(error);
    }
  }

  return {
    video: new ReadableStream({
      start(controller) {
        videoController = controller;
        pump();
      },
    }),
    audio: new ReadableStream({
      start(controller) {
        audioController = controller;
        pump();
      },
    }),
  };
}

gopeed.events.onResolve(async function (ctx) {
  if (ctx.req.url.includes("/single-fetch-stream")) {
    const payloadSize = Number(new URL(ctx.req.url).searchParams.get("size")) || 0;
    const url = gopeed.runtime.blob.createObjectURL(async () => createFetchDrainedStream(sourceURL(ctx.req.url)), {
      size: payloadSize,
    });
    ctx.res = {
      name: "single-fetch-stream",
      files: [
        {
          name: "single.bin",
          size: payloadSize,
          req: { url },
        },
      ],
    };
    return;
  }

  if (ctx.req.url.includes("/range-fetch-stream")) {
    const payloadSize = Number(new URL(ctx.req.url).searchParams.get("size")) || 0;
    const url = gopeed.runtime.blob.createObjectURL(
      async ({ offset = 0, end = -1 }) => createFetchDrainedRangeStream(sourceURL(ctx.req.url), offset, end),
      {
        size: payloadSize,
        range: true,
      }
    );
    ctx.res = {
      name: "range-fetch-stream",
      range: true,
      files: [
        {
          name: "range.bin",
          size: payloadSize,
          req: { url },
        },
      ],
    };
    return;
  }

  if (ctx.req.url.includes("/multiplex-fetch-stream")) {
    const videoChunks = ["video-0", "video-1", "video-2", "video-3"];
    const audioChunks = ["audio-0", "audio-1", "audio-2", "audio-3"];
    let streams;
    const openStreams = () => {
      if (!streams) {
        streams = createMultiplexStreams(sourceURL(ctx.req.url));
      }
      return streams;
    };
    const videoUrl = gopeed.runtime.blob.createObjectURL(async () => openStreams().video, {
      size: videoChunks.reduce((sum, chunk) => sum + byteLength(chunk), 0),
    });
    const audioUrl = gopeed.runtime.blob.createObjectURL(async () => openStreams().audio, {
      size: audioChunks.reduce((sum, chunk) => sum + byteLength(chunk), 0),
    });
    ctx.res = {
      name: "multiplex-fetch-stream",
      files: [
        {
          name: "video.bin",
          size: videoChunks.reduce((sum, chunk) => sum + byteLength(chunk), 0),
          req: { url: videoUrl },
        },
        {
          name: "audio.bin",
          size: audioChunks.reduce((sum, chunk) => sum + byteLength(chunk), 0),
          req: { url: audioUrl },
        },
      ],
    };
  }
});
