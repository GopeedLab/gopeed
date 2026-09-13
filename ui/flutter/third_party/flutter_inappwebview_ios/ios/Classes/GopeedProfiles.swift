// Internal Gopeed host bridge. No extension-facing API.
import Foundation
import WebKit
import Network
import Flutter

final class GopeedProfiles {
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
        let store = stores[identifier] ?? WKWebsiteDataStore(forIdentifier: uuid)
        let endpoint = nw_endpoint_create_host(host, String(port))
        let proxy = nw_proxy_config_create_socksv5(endpoint)
        store.__proxyConfigurations = [proxy]
        stores[identifier] = store
        result(true)
    }
}
