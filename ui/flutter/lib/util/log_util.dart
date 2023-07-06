import 'dart:io';
import 'package:flutter/foundation.dart';
import 'package:logger/logger.dart';
import 'util.dart';

final logger = Logger(
  filter: ProductionFilter(),
  printer: LogfmtPrinter(),
  output: buildOutput(),
);

buildOutput() {
  // if is debug mode, don't log to file
  if (!kDebugMode && Util.isDesktop()) {
    const logDirPath = 'logs';
    var logDir = Directory(logDirPath);
    if (!logDir.existsSync()) {
      logDir.createSync();
    }
    return FileOutput(file: File('$logDirPath/client.log'));
  }
  return null;
}
