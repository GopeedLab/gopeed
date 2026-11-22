import Cocoa
import FlutterMacOS
import desktop_multi_window
import window_manager

class MainFlutterWindow: NSWindow {
  private var childWindowCloseObservers: [ObjectIdentifier: NSObjectProtocol] = [:]

  override func awakeFromNib() {
    let flutterViewController = FlutterViewController()
    let windowFrame = self.frame
    self.contentViewController = flutterViewController
    self.setFrame(windowFrame, display: true)

    RegisterGeneratedPlugins(registry: flutterViewController)
    FlutterMultiWindowPlugin.setOnWindowCreatedCallback { [weak self] controller in
      RegisterGeneratedPlugins(registry: controller)
      self?.releaseChildEngineOnClose(controller)
    }

    super.awakeFromNib()
  }

  // desktop_multi_window removes closed windows from its registry, but plugin
  // channels can still retain the window/controller/engine cycle. NSWindow.close
  // alone therefore leaves each child Dart isolate and its heap running.
  private func releaseChildEngineOnClose(_ controller: FlutterViewController) {
    guard let window = controller.view.window else { return }
    let key = ObjectIdentifier(window)
    childWindowCloseObservers[key] = NotificationCenter.default.addObserver(
      forName: NSWindow.willCloseNotification, object: window, queue: .main
    ) { [weak self, weak controller, weak window] _ in
      if let observer = self?.childWindowCloseObservers.removeValue(forKey: key) {
        NotificationCenter.default.removeObserver(observer)
      }
      // Let close notifications and the plugin's registry removal finish before
      // tearing down the engine that delivered the close request.
      DispatchQueue.main.async { [weak controller, weak window] in
        // window_manager's onEvent closure retains its plugin, which retains
        // the manager and NSWindow even after the engine has stopped.
        (window?.delegate as? WindowManager)?.onEvent = nil
        window?.delegate = nil
        controller?.engine.shutDownEngine()
        window?.contentViewController = nil
      }
    }
  }

  deinit {
    for observer in childWindowCloseObservers.values {
      NotificationCenter.default.removeObserver(observer)
    }
  }

  override public func order(_ place: NSWindow.OrderingMode, relativeTo otherWin: Int) {
    super.order(place, relativeTo: otherWin)
    hiddenWindowAtLaunch()
  }
}
