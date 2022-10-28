import 'dart:io';

import 'package:logger/logger.dart';
import 'package:logger/src/outputs/file_output.dart';

import 'util.dart';

final logger = Logger(
  printer: PrettyPrinter(),
  output: Util.isDesktop() ? FileOutput(file: File('log.txt')) : null,
);
