import Cocoa
import FlutterMacOS
import app_links

@main
class AppDelegate: FlutterAppDelegate {
  public override func application(_ application: NSApplication,
                                  continue userActivity: NSUserActivity,
                                  restorationHandler: @escaping ([any NSUserActivityRestoring]) -> Void) -> Bool {

    guard let url = AppLinks.shared.getUniversalLink(userActivity) else {
      return false
    }
    
    AppLinks.shared.handleLink(link: url.absoluteString)
    
    return false // Returning true will stop the propagation to other packages
  }

  override func applicationShouldTerminateAfterLastWindowClosed(_ sender: NSApplication) -> Bool {
    return false
  }

  override func applicationShouldHandleReopen(_ sender: NSApplication, hasVisibleWindows flag: Bool) -> Bool {
    if !flag {
      for window in NSApp.windows {
        if !window.isVisible {
          window.setIsVisible(true)
        }
        window.makeKeyAndOrderFront(self)
        NSApp.activate(ignoringOtherApps: true)
      }
    }
    return true
  }

  override func application(_ sender: NSApplication, openFile filename: String) -> Bool {
    let url = URL(fileURLWithPath: filename)
    AppLinks.shared.handleLink(link: url.absoluteString)
    return false;
  }

  override func application(_ application: NSApplication, open urls: [URL]) {
    // Only handle the first file
    if(urls.isEmpty) {
      return
    }
    AppLinks.shared.handleLink(link: urls.first!.absoluteString)
  }
}
