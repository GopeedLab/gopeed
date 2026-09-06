import 'dart:io';

import 'package:dart_ipc/dart_ipc.dart';

Future<ServerSocket> bindIpc(String path) => bind(path);
