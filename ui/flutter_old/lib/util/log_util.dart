import 'dart:io';
import 'package:flutter/foundation.dart';
import 'package:logger/logger.dart';
import 'package:path/path.dart' as path;
import 'util.dart';

late final Logger logger;

initLogger() {
  // if is debug mode, don't log to file
  logger = Logger(
    filter: ProductionFilter(),
    printer: SimplePrinter(printTime: true, colors: false),
    output: _buildOutput(),
  );
}

String logsDir() {
  return path.join(Util.getStorageDir(), 'logs');
}

_buildOutput() {
  // if is debug mode, don't log to file
  if (!kDebugMode && Util.isDesktop()) {
    final logDirPath = logsDir();
    var logDir = Directory(logsDir());
    if (!logDir.existsSync()) {
      logDir.createSync();
    }
    return FileOutput(file: File('$logDirPath/client.log'));
  }
  return null;
}
