import 'dart:io';

import 'package:flutter/foundation.dart';
import 'package:logger/logger.dart';

import 'util.dart';

final logger = Logger(
  printer: PrettyPrinter(),
  output: buildOutput(),
);

buildOutput() {
  // if is debug mode, output to console
  if (!kDebugMode && Util.isDesktop()) {
    return FileOutput(file: File('log.txt'));
  }
  return null;
}
