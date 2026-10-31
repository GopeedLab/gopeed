# Batch task creation

`POST /api/v1/tasks/batch` accepts ordinary and grouped batches:

```json
{
  "group": true,
  "groupName": "album",
  "opts": {"path": "/downloads"},
  "reqs": [
    {"rid": "resolved-resource-id", "opts": {"selectFiles": [1]}},
    {"req": {"url": "https://example.com/cover.jpg"}}
  ]
}
```

Each member accepts `rid` or `req`. A nonempty `rid` takes priority and creates a
task from the cached resolve result. A member's `opts`, when present, replaces
batch `opts` as a whole; omitted fields do not inherit individual batch values.
Members without `opts` use batch `opts`. BT file selections use the indexes from
the resolve result. Requests without `rid` continue to work.

For groups, batch `opts.path` selects the parent directory. Members inheriting
batch options, or with an empty path, use the reserved group directory. Explicit
relative member paths are resolved under that directory; do not prefix them with
the batch parent path. For example, parent `downloads` and member path `nested`
place the member in `downloads/<group name>/nested`. Absolute paths under the
parent directory preserve their relative structure within the group. In ordinary
batches, an empty member path uses the configured default download directory.

All members are attempted. If at least one task is persisted, the response is
successful and contains the created task IDs. A task that fails to start after
being persisted is included; use the existing task controls to retry it.
Successful tasks are kept when other members fail. If every member fails, the
API returns an error and removes any newly reserved empty group and directory.

Both ordinary and grouped batches now return an object in `data`:

```json
{"code": 0, "msg": "", "data": {"taskIds": ["task-1", "task-2"], "parentId": "group-id"}}
```

Ordinary batches omit `parentId`. **This replaces the former array response for
ordinary batches.** The outer `code`/`msg`/`data` envelope is unchanged. Clients
must read `data.taskIds`; there are no per-member results. A successful response
completes the batch, including partial success, so clients must not automatically
resubmit missing members.

With `group: true`, the server generates the group ID and reserves its directory.
Creation does not accept a client group ID or append to an existing group. Group
metadata remains available through the existing task list and detail responses;
`parentId` identifies membership for group operations.
