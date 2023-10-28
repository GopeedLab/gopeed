gopeed.events.onResolve(async function (ctx) {
    const key = "key"
    const value1 = "value1", value2 = JSON.stringify({a: 1, b: "2"})

    if (gopeed.storage.get(key) !== null) {
        throw new Error("storage get null error")
    }
    gopeed.storage.remove(key)
    if(gopeed.storage.keys().length !== 0) {
        throw new Error("storage keys null error")
    }

    gopeed.storage.set(key, value1)
    if (gopeed.storage.get(key) !== value1) {
        throw new Error("storage put1 error")
    }

    gopeed.storage.set(key, value2)
    if (gopeed.storage.get(key) !== value2) {
        throw new Error("storage put2 error")
    }

    if(gopeed.storage.keys().length !== 1) {
        throw new Error("storage keys error")
    }

    gopeed.storage.remove(key)
    if (gopeed.storage.get(key) !== null) {
        throw new Error("storage delete error")
    }

    gopeed.storage.set(key, value1)
    gopeed.storage.clear()
    if (gopeed.storage.get(key) !== null) {
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
