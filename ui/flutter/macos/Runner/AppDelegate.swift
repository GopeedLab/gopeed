import Cocoa
import FlutterMacOS
import app_links

@main
class AppDelegate: FlutterAppDelegate {
  override func application(
    _ application: NSApplication,
    continue userActivity: NSUserActivity,
    restorationHandler: @escaping ([any NSUserActivityRestoring]) -> Void
  ) -> Bool {
    guard let url = AppLinks.shared.getUniversalLink(userActivity) else {
      return false
    }
    AppLinks.shared.handleLink(link: url.absoluteString)
    // Keep propagating the activity to other registered plugins.
    return false
  }

  override func applicationShouldTerminateAfterLastWindowClosed(_ sender: NSApplication) -> Bool {
    return false
  }

  override func applicationSupportsSecureRestorableState(_ app: NSApplication) -> Bool {
    return true
  }

  override func applicationShouldHandleReopen(_ sender: NSApplication, hasVisibleWindows flag: Bool) -> Bool {
    if !flag {
      for window in NSApp.windows {
        if !window.isVisible {
          window.setIsVisible(true)
        }
        window.makeKeyAndOrderFront(self)
      }
      NSApp.activate(ignoringOtherApps: true)
    }
    return true
  }

  override func application(_ sender: NSApplication, openFile filename: String) -> Bool {
    AppLinks.shared.handleLink(link: URL(fileURLWithPath: filename).absoluteString)
    return true
  }

  override func application(_ application: NSApplication, open urls: [URL]) {
    guard let url = urls.first else { return }
    AppLinks.shared.handleLink(link: url.absoluteString)
  }
}
