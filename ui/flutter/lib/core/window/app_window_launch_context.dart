import 'package:desktop_multi_window/desktop_multi_window.dart';

import 'app_window_payload.dart';

class AppWindowLaunchContext {
  const AppWindowLaunchContext._({required this.isSubWindow, required this.payload, this.windowId});

  factory AppWindowLaunchContext.main() {
    return AppWindowLaunchContext._(isSubWindow: false, payload: AppWindowPayload.main());
  }

  factory AppWindowLaunchContext.subWindow({required String windowId, required AppWindowPayload payload}) {
    return AppWindowLaunchContext._(isSubWindow: true, windowId: windowId, payload: payload);
  }

  factory AppWindowLaunchContext.fromArgs(List<String> args) {
    if (args.length >= 2 && args.first == 'multi_window') {
      final windowId = args[1];
      final payload = args.length >= 3 ? AppWindowPayload.fromRaw(args[2]) : AppWindowPayload.main();
      return AppWindowLaunchContext.subWindow(windowId: windowId, payload: payload);
    }

    return AppWindowLaunchContext.main();
  }

  final bool isSubWindow;
  final String? windowId;
  final AppWindowPayload payload;

  WindowController? buildController() {
    if (!isSubWindow || windowId == null) {
      return null;
    }
    return WindowController.fromWindowId(windowId!);
  }
}
