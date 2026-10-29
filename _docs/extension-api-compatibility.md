# Extension request API compatibility

The request mutation API introduced in #1538 remains available at
`task.meta.req` in `onStart` and `onError`. The legacy APIs below are retained
for compatibility during 2.x and are deprecated rather than removed.
Any removal should happen in a future major release with migration notice.

| Legacy API | Preferred API |
| --- | --- |
| `await ctx.task.setUrl(url)` in `onStart` / `onError` | `await ctx.task.meta.req.setUrl(url)` |
| `await ctx.req.setLabels(labels)` in `onResolve` | `ctx.req.labels = labels` |
| `await ctx.req.putLabel(key, value)` in `onResolve` | Initialize `ctx.req.labels` if needed, then assign `ctx.req.labels[key] = value` |
| `await ctx.req.delLabel(key)` in `onResolve` | If labels exist, `delete ctx.req.labels[key]` |

The three legacy label methods also remain available on
`ctx.task.meta.req` in `onDone`; prefer direct label edits there.
`setLabels` replaces all labels, `putLabel` preserves other labels and
initializes missing labels, and `delLabel` ignores missing keys.

In `onStart` and `onError`, the supported `task.meta.req.setLabels`,
`putLabel`, and `delLabel` methods remain the preferred API and are not
deprecated. The old task-level `setUrl` forwards to the same request as
`task.meta.req.setUrl`, so both forms can be used during migration.

Event capabilities stay the same: `task.continue()` is available only in
`onError`, and task-level `setUrl` is available only in `onStart` and
`onError`. This compatibility change does not introduce `task.pause()`.
