gopeed.events.onResolve(async function (ctx) {
    const key = "key"
    const value1 = "value1", value2 = JSON.stringify({a: 1, b: "2"})

    if (ctx.storage.get(key) !== null) {
        throw new Error("storage get null error")
    }
    ctx.storage.remove(key)
    if(ctx.storage.keys().length !== 0) {
        throw new Error("storage keys null error")
    }

    ctx.storage.set(key, value1)
    if (ctx.storage.get(key) !== value1) {
        throw new Error("storage put1 error")
    }

    ctx.storage.set(key, value2)
    if (ctx.storage.get(key) !== value2) {
        throw new Error("storage put2 error")
    }

    if(ctx.storage.keys().length !== 1) {
        throw new Error("storage keys error")
    }

    ctx.storage.remove(key)
    if (ctx.storage.get(key) !== null) {
        throw new Error("storage delete error")
    }

    ctx.storage.set(key, value1)
    ctx.storage.clear()
    if (ctx.storage.get(key) !== null) {
        throw new Error("storage clear error")
    }

    ctx.res = {
        name: "test",
        files: Array(2).fill(true).map((_, i) => ({
                name: `test-${i}.txt`,
                size: 1024,
                req: {
                    url: ctx.req.url + "/" + i,
                }
            }),
        ),
    };
});
