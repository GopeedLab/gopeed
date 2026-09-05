import 'package:flutter/foundation.dart';

abstract final class AppWindowChrome {
  static bool get isDesktopWindow =>
      !kIsWeb &&
      (defaultTargetPlatform == TargetPlatform.windows ||
          defaultTargetPlatform == TargetPlatform.macOS ||
          defaultTargetPlatform == TargetPlatform.linux);

  static bool get usesCustomChrome =>
      !kIsWeb && (defaultTargetPlatform == TargetPlatform.windows || defaultTargetPlatform == TargetPlatform.linux);

  static bool get reservesHeaderInset => reservesHeaderInsetFor(web: kIsWeb, platform: defaultTargetPlatform);

  static bool reservesHeaderInsetFor({required bool web, required TargetPlatform platform}) {
    return web ||
        platform == TargetPlatform.windows ||
        platform == TargetPlatform.macOS ||
        platform == TargetPlatform.linux;
  }

  static bool get clipsRoundedCorners => !kIsWeb && defaultTargetPlatform == TargetPlatform.windows;
}
