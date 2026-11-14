#ifndef RUNNER_CRASH_HANDLER_H_
#define RUNNER_CRASH_HANDLER_H_

#include <windows.h>

// Installs best-effort native diagnostics before Flutter or plugins start.
void InstallCrashHandler();

// Records the exception and continues normal Windows exception processing.
LONG RecordNativeCrash(EXCEPTION_POINTERS* exception);

#endif  // RUNNER_CRASH_HANDLER_H_
