// Internal Gopeed host bridge. No extension-facing API.
import Foundation
import WebKit
import Network
import Flutter

final class GopeedProfiles {
    private static var proxyURLs: [String: String] = [:]
    private static var stores: [String: WKWebsiteDataStore] = [:]
    static func store(_ identifier: String) -> WKWebsiteDataStore? { stores[identifier] }

    static func prepare(arguments: NSDictionary?, result: @escaping FlutterResult) {
        guard #available(macOS 14.0, iOS 17.0, *),
              let identifier = arguments?["gopeedProfileId"] as? String,
              let uuid = UUID(uuidString: identifier),
              let proxyURL = arguments?["proxyUrl"] as? String,
              let url = URL(string: proxyURL), url.scheme == "socks5",
              let host = url.host, host == "127.0.0.1",
              let port = url.port, port > 0 && port <= 65535 else {
            result(FlutterError(code: "UNAVAILABLE", message: "Isolated WebView profiles and proxy require macOS 14 / iOS 17 or newer and valid host configuration", details: nil))
            return
        }
        if stores[identifier] != nil && proxyURLs[identifier] == proxyURL {
            result(true)
            return
        }
        let store = stores[identifier] ?? WKWebsiteDataStore(forIdentifier: uuid)
        let endpoint = nw_endpoint_create_host(host, String(port))
        let proxy = nw_proxy_config_create_socksv5(endpoint)
        store.__proxyConfigurations = [proxy]
        stores[identifier] = store
        proxyURLs[identifier] = proxyURL
        result(true)
    }
    static func remove(arguments: NSDictionary?, result: @escaping FlutterResult) {
        guard let identifier = arguments?["gopeedProfileId"] as? String,
              let uuid = UUID(uuidString: identifier) else {
            result(FlutterError(code: "INVALID_REQUEST", message: "Invalid WebView profile identifier", details: nil))
            return
        }
        guard #available(macOS 14.0, iOS 17.0, *) else {
            // Named stores could not have been created on these OS versions.
            result(true)
            return
        }
        // Deleting a named store alone can leave in-process cookie caches.
        // Clear every website data type before releasing/removing the store.
        let store = stores[identifier] ?? WKWebsiteDataStore(forIdentifier: uuid)
        store.removeData(ofTypes: WKWebsiteDataStore.allWebsiteDataTypes(), modifiedSince: Date.distantPast) {
            stores.removeValue(forKey: identifier)
            proxyURLs.removeValue(forKey: identifier)
            removeStore(uuid, attempts: 100, result: result)
        }
    }

    @available(macOS 14.0, iOS 17.0, *)
    private static func removeStore(_ uuid: UUID, attempts: Int, result: @escaping FlutterResult) {
        WKWebsiteDataStore.remove(forIdentifier: uuid) { error in
            if error != nil && attempts > 1 {
                // Network-process references can outlive platform-view disposal.
                DispatchQueue.main.asyncAfter(deadline: .now() + 0.1) {
                    removeStore(uuid, attempts: attempts - 1, result: result)
                }
                return
            }
            if let error = error {
                result(FlutterError(code: "PROFILE_REMOVE_FAILED", message: error.localizedDescription, details: nil))
            } else {
                result(true)
            }
        }
    }

}
