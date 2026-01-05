import 'arch_stub.dart'
    if (dart.library.io) 'entry/arch_native.dart'
    if (dart.library.html) 'entry/arch_web.dart';

// Copy from pkg/sky_engine/lib/ffi/abi.dart
enum Architecture {
  arm,
  arm64,
  ia32,
  x64,
  riscv32,
  riscv64,
}

Architecture getArch() => doGetArch();
