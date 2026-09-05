import 'arch_stub.dart' if (dart.library.io) 'entry/arch_native.dart' if (dart.library.html) 'entry/arch_web.dart';

// These names match the architecture suffixes used by Gopeed release assets.
enum Architecture { arm, arm64, ia32, x64, riscv32, riscv64 }

Architecture getArch() => doGetArch();
