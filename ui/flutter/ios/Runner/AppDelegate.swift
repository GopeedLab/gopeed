import Flutter
import Libgopeed
import UIKit

private final class GopeedTaskEventForwarder: NSObject, LibgopeedTaskEventListener {
  private let channel: FlutterMethodChannel

  init(channel: FlutterMethodChannel) {
    self.channel = channel
  }

  func onTaskEvent(_ payload: String?) {
    DispatchQueue.main.async { [channel] in
      channel.invokeMethod("taskEvent", arguments: payload ?? "")
    }
  }
}

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
    let taskEventForwarder = GopeedTaskEventForwarder(channel: libgopeedChannel)
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
      case "getApiServerState":
        result(LibgopeedGetAPIServerState())
      case "startApiServer":
        result(LibgopeedStartAPIServer())
      case "stopApiServer":
        result(LibgopeedStopAPIServer())
      case "restartApiServer":
        result(LibgopeedRestartAPIServer())
      case "invoke":
        let arguments = call.arguments as? [String: Any]
        let method = arguments?["method"] as? String ?? ""
        let path = arguments?["path"] as? String ?? ""
        let query = arguments?["query"] as? String ?? ""
        let body = arguments?["body"] as? String ?? ""
        result(LibgopeedInvoke(method, path, query, body))
      case "subscribeTaskEvents":
        let arguments = call.arguments as? [String: Any]
        let mask = (arguments?["mask"] as? NSNumber)?.int64Value ?? 0
        LibgopeedSubscribeTaskEvents(mask, mask == 0 ? nil : taskEventForwarder)
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
