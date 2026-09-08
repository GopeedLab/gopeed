import Cocoa
import FlutterMacOS
import XCTest
@testable import Gopeed

class RunnerTests: XCTestCase {

  func testReopenDoesNotRestoreClosedCreateTaskWindows() throws {
    let mainWindow = try XCTUnwrap(NSApp.windows.first { $0 is MainFlutterWindow })
    let delegate = try XCTUnwrap(NSApp.delegate as? AppDelegate)
    let children = [makeChildWindow(), makeChildWindow()]
    defer {
      children.forEach { $0.close() }
      mainWindow.makeKeyAndOrderFront(nil)
    }

    mainWindow.orderOut(nil)
    for child in children {
      child.makeKeyAndOrderFront(nil)
      child.close()
      XCTAssertFalse(child.isVisible)
    }

    XCTAssertFalse(delegate.applicationShouldHandleReopen(NSApp, hasVisibleWindows: false))
    XCTAssertTrue(mainWindow.isVisible)
    XCTAssertTrue(children.allSatisfy { !$0.isVisible })
  }

  func testReopenRestoresMainWindowWithAnUnfinishedChildVisible() throws {
    let mainWindow = try XCTUnwrap(NSApp.windows.first { $0 is MainFlutterWindow })
    let delegate = try XCTUnwrap(NSApp.delegate as? AppDelegate)
    let child = makeChildWindow()
    defer {
      child.close()
      mainWindow.makeKeyAndOrderFront(nil)
    }

    mainWindow.orderOut(nil)
    child.makeKeyAndOrderFront(nil)

    XCTAssertFalse(delegate.applicationShouldHandleReopen(NSApp, hasVisibleWindows: true))
    XCTAssertTrue(mainWindow.isVisible)
    XCTAssertTrue(child.isVisible)
  }

  private func makeChildWindow() -> NSWindow {
    let window = NSWindow(
      contentRect: NSRect(x: 0, y: 0, width: 300, height: 200),
      styleMask: [.titled, .closable], backing: .buffered, defer: false)
    // Match desktop_multi_window's retained native child windows.
    window.isReleasedWhenClosed = false
    return window
  }

}
