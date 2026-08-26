import 'dart:ffi';

import '../arch.dart';

Architecture doGetArch() {
  final archName = Abi.current().toString().split('_')[1];
  return Architecture.values.firstWhere(
    (architecture) => architecture.name == archName,
    orElse: () => Architecture.x64,
  );
}
