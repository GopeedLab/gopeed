import UIKit
import Flutter
import Libgopeed

@UIApplicationMain
@objc class AppDelegate: FlutterAppDelegate {
    override func application(
        _ application: UIApplication,
        didFinishLaunchingWithOptions launchOptions: [UIApplication.LaunchOptionsKey: Any]?
    ) -> Bool {
        let controller : FlutterViewController = window?.rootViewController as! FlutterViewController
        let batteryChannel = FlutterMethodChannel(name: "gopeed.com/libgopeed",
                                                  binaryMessenger: controller.binaryMessenger)
        batteryChannel.setMethodCallHandler({
            (call: FlutterMethodCall, result: @escaping FlutterResult) -> Void in
            switch call.method {
            case "start":
                let args = call.arguments as? Dictionary<String, Any>
                let cfg = args?["cfg"] as? String
                let portPrt = UnsafeMutablePointer<Int>.allocate(capacity: MemoryLayout<Int>.stride)
                var error: NSError?
                if LibgopeedStart(cfg, portPrt, &error){
                    result(portPrt.pointee)
                }else{
                    result(FlutterError(code: "ERROR", message: error.debugDescription, details: nil))
                }
            case "stop":
                LibgopeedStop()
                result(nil)
            default:
                result(FlutterMethodNotImplemented)
            }
        })
        
        GeneratedPluginRegistrant.register(with: self)

        SwiftFlutterForegroundTaskPlugin.setPluginRegistrantCallback(registerPlugins)
        if #available(iOS 10.0, *) {
            UNUserNotificationCenter.current().delegate = self as? UNUserNotificationCenterDelegate
        }
        return super.application(application, didFinishLaunchingWithOptions: launchOptions)
    }
}

func registerPlugins(registry: FlutterPluginRegistry) {
  GeneratedPluginRegistrant.register(with: registry)
}
