// ignore: avoid_web_libraries_in_flutter
import 'dart:ffi';

import '../arch.dart';

Architecture doGetArch() {
  final currentAbi = Abi.current().toString();
  final archName = currentAbi.split("_")[1];
  final arch = Architecture.values.firstWhere(
      (element) => element.name == archName,
      orElse: () => Architecture.x64);
  return arch;
}
