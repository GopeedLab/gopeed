import 'dart:io';

import 'package:flutter/foundation.dart';

TargetPlatform get downloadDirectoryHostPlatform {
  if (Platform.isAndroid) return TargetPlatform.android;
  if (Platform.isIOS) return TargetPlatform.iOS;
  if (Platform.isWindows) return TargetPlatform.windows;
  if (Platform.isLinux) return TargetPlatform.linux;
  if (Platform.isMacOS) return TargetPlatform.macOS;
  return defaultTargetPlatform;
}
