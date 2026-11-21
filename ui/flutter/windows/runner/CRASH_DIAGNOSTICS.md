# Windows native crash diagnostics

The runner installs diagnostics before starting Flutter and plugins. Files are
written beside the executable under `storage/logs`, matching Windows app storage:

- `native-crash.log`: UTC timestamp, app version, process/thread, exception code,
  faulting module/address/offset, and minidump success or the Windows error code.
- `native-crash.dmp`: the most recent native crash, including thread stacks,
  loaded modules and unloaded module information.
- `crash.log`: Go runtime crash output, kept separately from native diagnostics.

Logs are appended and existing dumps survive normal restarts. A subsequent native
crash replaces the previous dump. Dump generation is best effort, particularly
for stack exhaustion, heap corruption or unavailable disk space. Process kills
and fail-fast termination can bypass exception handlers.

The top-level unhandled filter covers native exceptions on process threads. An
SEH boundary inside the runner's window callback records the original escaping exception
before Windows wraps a window callback failure as `0xC000041D`. A vectored handler
also observes that fatal callback status, since it can bypass the unhandled
filter. All handlers continue exception processing; they do not attempt recovery
or log ordinary handled first-chance exceptions.

On Windows, verify a release build by exercising tray clicks, Explorer restart,
system font changes and child-window open/close. In a disposable test build,
raise an unhandled access violation inside a window callback and on a worker
thread, then raise `0xC000041D`; check that the log and dump are written and
survive relaunch. A debugger may intercept these before application handlers.
Open the dump with WinDbg and matching runner/Flutter/plugin PDBs to locate the
faulting stack. Minidumps can contain application data; inspect them before
sharing.
