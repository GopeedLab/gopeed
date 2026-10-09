import Cocoa
import FlutterMacOS
import desktop_multi_window
import window_manager

class MainFlutterWindow: NSWindow {
  // Keep this in sync with AppDesignTokens.railWidth in Flutter.
  private static let navigationRailWidth: CGFloat = 68
  private static let trafficLightGap: CGFloat = 6
  private var trafficLightLayoutScheduled = false
  private var fullScreenTransitionInProgress = false

  override func awakeFromNib() {
    let flutterViewController = FlutterViewController()
    let windowFrame = self.frame
    self.contentViewController = flutterViewController
    self.setFrame(windowFrame, display: true)

    RegisterGeneratedPlugins(registry: flutterViewController)
    FlutterMultiWindowPlugin.setOnWindowCreatedCallback { controller in
      RegisterGeneratedPlugins(registry: controller)
    }

    super.awakeFromNib()

    for name in [NSWindow.didResizeNotification, NSWindow.didBecomeKeyNotification,
                 NSWindow.didChangeBackingPropertiesNotification] {
      NotificationCenter.default.addObserver(
        self, selector: #selector(refreshTrafficLightLayout(_:)), name: name, object: self)
    }
    for name in [NSWindow.willEnterFullScreenNotification, NSWindow.willExitFullScreenNotification] {
      NotificationCenter.default.addObserver(
        self, selector: #selector(beginFullScreenTransition(_:)), name: name, object: self)
    }
    for name in [NSWindow.didEnterFullScreenNotification, NSWindow.didExitFullScreenNotification] {
      NotificationCenter.default.addObserver(
        self, selector: #selector(endFullScreenTransition(_:)), name: name, object: self)
    }
    scheduleTrafficLightLayout()
  }

  private func positionTrafficLights() {
    // Let AppKit position controls in its own full-screen title bar.
    guard !styleMask.contains(.fullScreen) else { return }

    let buttons = [NSWindow.ButtonType.closeButton, .miniaturizeButton, .zoomButton]
      .compactMap { standardWindowButton($0) }
    guard buttons.count == 3 else { return }

    let width = buttons.reduce(CGFloat.zero) { $0 + $1.frame.width }
      + Self.trafficLightGap * CGFloat(buttons.count - 1)
    var x = (Self.navigationRailWidth - width) / 2
    guard x >= 0 else { return }

    // Keep native sizes, vertical positions and button behavior intact.
    for button in buttons {
      var origin = button.frame.origin
      origin.x = x
      if button.frame.origin != origin {
        button.setFrameOrigin(origin)
      }
      x += button.frame.width + Self.trafficLightGap
    }
  }

  @objc private func refreshTrafficLightLayout(_ notification: Notification) {
    scheduleTrafficLightLayout()
  }

  @objc private func beginFullScreenTransition(_ notification: Notification) {
    fullScreenTransitionInProgress = true
  }

  @objc private func endFullScreenTransition(_ notification: Notification) {
    fullScreenTransitionInProgress = false
    scheduleTrafficLightLayout()
  }

  private func scheduleTrafficLightLayout() {
    guard !trafficLightLayoutScheduled else { return }
    trafficLightLayoutScheduled = true
    // AppKit can reset button frames while finishing a resize or style change.
    DispatchQueue.main.async { [weak self] in
      guard let self = self else { return }
      self.trafficLightLayoutScheduled = false
      guard !self.fullScreenTransitionInProgress, !self.styleMask.contains(.fullScreen) else { return }
      // Calling this is supported; overriding AppKit's layout entry point is not.
      self.layoutIfNeeded()
      self.positionTrafficLights()
    }
  }

  deinit {
    NotificationCenter.default.removeObserver(self)
  }

  override public func order(_ place: NSWindow.OrderingMode, relativeTo otherWin: Int) {
    super.order(place, relativeTo: otherWin)
    hiddenWindowAtLaunch()
    if place == .above {
      scheduleTrafficLightLayout()
    }
  }
}
