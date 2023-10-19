gopeed.events.onResolve(async function (ctx) {
    if (gopeed.settings.string != null) {
        throw new Error("string is not null");
    }
    if (gopeed.settings.number != null) {
        throw new Error("number is not null");
    }
    if (gopeed.settings.boolean != null) {
        throw new Error("boolean is not null");
    }

    if (gopeed.settings.stringDefault !== "default") {
        throw new Error("string default value is incorrect");
    }
    if (gopeed.settings.numberDefault !== 1) {
        throw new Error("number default value is incorrect");
    }
    if (gopeed.settings.booleanDefault !== true) {
        throw new Error("boolean default value is incorrect");
    }

    if (gopeed.settings.stringValued !== "valued") {
        throw new Error("string value is incorrect");
    }
    if (gopeed.settings.numberValued !== 1.1) {
        throw new Error("number value is incorrect");
    }
    if (gopeed.settings.booleanValued !== true) {
        throw new Error("boolean value is incorrect");
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
