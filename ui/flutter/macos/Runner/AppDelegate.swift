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
    // Closed child windows can remain in NSApp.windows. Restoring every
    // window resurrects completed create-task forms with their last frame.
    // Use the main window even when an unfinished child is still visible.
    if let window = sender.windows.first(where: { $0 is MainFlutterWindow }) {
      if window.isMiniaturized {
        window.deminiaturize(self)
      }
      window.makeKeyAndOrderFront(self)
      sender.activate(ignoringOtherApps: true)
    }
    // We handled reopening; AppKit must not restore other retained windows.
    return false
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
