import Flutter
import Libgopeed
import UIKit

@main
@objc class AppDelegate: FlutterAppDelegate, FlutterImplicitEngineDelegate {
  override func application(
    _ application: UIApplication,
    didFinishLaunchingWithOptions launchOptions: [UIApplication.LaunchOptionsKey: Any]?
  ) -> Bool {
    return super.application(application, didFinishLaunchingWithOptions: launchOptions)
  }

  func didInitializeImplicitFlutterEngine(_ engineBridge: FlutterImplicitEngineBridge) {
    GeneratedPluginRegistrant.register(with: engineBridge.pluginRegistry)
    let messenger = engineBridge.applicationRegistrar.messenger()

    let libgopeedChannel = FlutterMethodChannel(
      name: "gopeed.com/libgopeed",
      binaryMessenger: messenger
    )
    libgopeedChannel.setMethodCallHandler { call, result in
      switch call.method {
      case "start":
        let arguments = call.arguments as? [String: Any]
        let config = arguments?["cfg"] as? String
        let port = UnsafeMutablePointer<Int>.allocate(capacity: 1)
        defer { port.deallocate() }
        var error: NSError?
        if LibgopeedStart(config, port, &error) {
          result(port.pointee)
        } else {
          result(
            FlutterError(
              code: "ERROR",
              message: error?.localizedDescription ?? "Failed to start Libgopeed",
              details: nil
            )
          )
        }
      case "stop":
        LibgopeedStop()
        result(nil)
      default:
        result(FlutterMethodNotImplemented)
      }
    }

    let locationChannel = FlutterMethodChannel(
      name: "gopeed/location_keep_alive",
      binaryMessenger: messenger
    )
    locationChannel.setMethodCallHandler { call, result in
      switch call.method {
      case "requestPermission":
        LocationKeepAliveManager.shared.requestPermission { authorized in
          result(authorized)
        }
      case "start":
        LocationKeepAliveManager.shared.start()
        result(nil)
      case "stop":
        LocationKeepAliveManager.shared.stop()
        result(nil)
      default:
        result(FlutterMethodNotImplemented)
      }
    }
  }
}
