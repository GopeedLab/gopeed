#include "crash_handler.h"

#include <dbghelp.h>
#include <strsafe.h>

#include <cstdint>
#include <cstring>
#include <filesystem>
#include <string>
#include <system_error>

namespace {

HANDLE crash_log = INVALID_HANDLE_VALUE;
std::wstring dump_path;
volatile LONG recording = 0;
LPTOP_LEVEL_EXCEPTION_FILTER previous_filter = nullptr;

void WriteLog(const char* text) {
  if (crash_log == INVALID_HANDLE_VALUE) return;
  DWORD written = 0;
  WriteFile(crash_log, text, static_cast<DWORD>(strlen(text)), &written,
            nullptr);
  FlushFileBuffers(crash_log);
}

LONG WINAPI UnhandledException(EXCEPTION_POINTERS* exception) {
  RecordNativeCrash(exception);
  return previous_filter ? previous_filter(exception)
                         : EXCEPTION_CONTINUE_SEARCH;
}

LONG CALLBACK FatalCallbackException(EXCEPTION_POINTERS* exception) {
  // Windows can terminate on this callback exception without invoking the
  // process's unhandled filter. Ignore ordinary first-chance exceptions (Go,
  // Dart and plugins may handle those) and observe only this fatal status.
  if (exception->ExceptionRecord->ExceptionCode == 0xC000041D) {
    RecordNativeCrash(exception);
  }
  return EXCEPTION_CONTINUE_SEARCH;
}

}  // namespace

LONG RecordNativeCrash(EXCEPTION_POINTERS* exception) {
  if (!exception || InterlockedCompareExchange(&recording, 1, 0) != 0) {
    return EXCEPTION_CONTINUE_SEARCH;
  }

  SYSTEMTIME time;
  GetSystemTime(&time);
  const auto* record = exception->ExceptionRecord;
  HMODULE module = nullptr;
  wchar_t module_path[MAX_PATH] = L"unknown";
  GetModuleHandleExW(GET_MODULE_HANDLE_EX_FLAG_FROM_ADDRESS |
                        GET_MODULE_HANDLE_EX_FLAG_UNCHANGED_REFCOUNT,
                    reinterpret_cast<LPCWSTR>(record->ExceptionAddress),
                    &module);
  if (module) GetModuleFileNameW(module, module_path, MAX_PATH);

  char line[2048] = {};
  StringCchPrintfA(
      line, ARRAYSIZE(line),
      "\r\n%04u-%02u-%02uT%02u:%02u:%02uZ Gopeed %s pid=%lu tid=%lu "
      "exception=0x%08lX address=%p module=%ls offset=0x%llX\r\n",
      time.wYear, time.wMonth, time.wDay, time.wHour, time.wMinute,
      time.wSecond, FLUTTER_VERSION, GetCurrentProcessId(), GetCurrentThreadId(),
      record->ExceptionCode, record->ExceptionAddress, module_path,
      static_cast<unsigned long long>(
          reinterpret_cast<uintptr_t>(record->ExceptionAddress) -
          reinterpret_cast<uintptr_t>(module)));
  WriteLog(line);

  HANDLE dump = CreateFileW(dump_path.c_str(), GENERIC_WRITE, 0, nullptr,
                            CREATE_ALWAYS, FILE_ATTRIBUTE_NORMAL, nullptr);
  if (dump != INVALID_HANDLE_VALUE) {
    MINIDUMP_EXCEPTION_INFORMATION info = {};
    info.ThreadId = GetCurrentThreadId();
    info.ExceptionPointers = exception;
    info.ClientPointers = FALSE;
    const BOOL ok = MiniDumpWriteDump(
        GetCurrentProcess(), GetCurrentProcessId(), dump,
        static_cast<MINIDUMP_TYPE>(MiniDumpNormal | MiniDumpWithThreadInfo |
                                   MiniDumpWithUnloadedModules),
        &info, nullptr, nullptr);
    const DWORD error = ok ? ERROR_SUCCESS : GetLastError();
    FlushFileBuffers(dump);
    CloseHandle(dump);
    StringCchPrintfA(line, ARRAYSIZE(line), "minidump success=%d error=%lu\r\n",
                     ok, error);
    WriteLog(line);
  } else {
    StringCchPrintfA(line, ARRAYSIZE(line), "minidump open error=%lu\r\n",
                     GetLastError());
    WriteLog(line);
  }

  // Do not swallow an exception or resume a potentially corrupted process.
  return EXCEPTION_CONTINUE_SEARCH;
}

void InstallCrashHandler() {
  // Match Dart's Windows storage path independently of the working directory.
  wchar_t executable[32768] = {};
  const DWORD length = GetModuleFileNameW(nullptr, executable, ARRAYSIZE(executable));
  if (length == 0 || length >= ARRAYSIZE(executable)) return;
  const auto directory = std::filesystem::path(executable).parent_path() /
                         L"storage" / L"logs";
  std::error_code error;
  std::filesystem::create_directories(directory, error);
  if (error) return;

  dump_path = (directory / L"native-crash.dmp").wstring();
  crash_log = CreateFileW((directory / L"native-crash.log").c_str(),
                         FILE_APPEND_DATA, FILE_SHARE_READ, nullptr, OPEN_ALWAYS,
                         FILE_ATTRIBUTE_NORMAL, nullptr);
  previous_filter = SetUnhandledExceptionFilter(UnhandledException);
  AddVectoredExceptionHandler(1, FatalCallbackException);
}
