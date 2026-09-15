//
//  CredentialDatabase.swift
//  flutter_inappwebview
//
//  Created by Lorenzo Pichilli on 29/10/2019.
//

import Foundation
import FlutterMacOS

public class CredentialDatabase: ChannelDelegate {
    static let METHOD_CHANNEL_NAME = "com.pichillilorenzo/flutter_inappwebview_credential_database"
    var plugin: InAppWebViewFlutterPlugin?
    static var credentialStore = URLCredentialStorage.shared

    init(plugin: InAppWebViewFlutterPlugin) {
        super.init(channel: FlutterMethodChannel(name: CredentialDatabase.METHOD_CHANNEL_NAME, binaryMessenger: plugin.registrar!.messenger))
        self.plugin = plugin
    }

    public override func handle(_ call: FlutterMethodCall, result: @escaping FlutterResult) {
        let arguments = call.arguments as? NSDictionary
        switch call.method {
            case "getAllAuthCredentials":
                var allCredentials: [[String: Any?]] = []

                for (protectionSpace, credentials) in CredentialDatabase.credentialStore.allCredentials {
                    var crendentials: [[String: Any?]] = []
                    for c in credentials {
                        let credential: [String: Any?] = c.value.toMap()
                        crendentials.append(credential)
                    }
                    if crendentials.count > 0 {
                        let dict: [String : Any] = [
                            "protectionSpace": protectionSpace.toMap(),
                            "credentials": crendentials
                        ]
                        allCredentials.append(dict)
                    }
                }
                result(allCredentials)
                break
            case "getHttpAuthCredentials":
                var crendentials: [[String: Any?]] = []

                let host = arguments!["host"] as! String
                let urlProtocol = arguments!["protocol"] as? String
                let urlPort = arguments!["port"] as? Int ?? 0
                var realm = arguments!["realm"] as? String;
                if let r = realm, r.isEmpty {
                    realm = nil
                }

                for (protectionSpace, credentials) in CredentialDatabase.credentialStore.allCredentials {
                    if protectionSpace.host == host && protectionSpace.realm == realm &&
                    protectionSpace.protocol == urlProtocol && protectionSpace.port == urlPort {
                        for c in credentials {
                            crendentials.append(c.value.toMap())
                        }
                        break
                    }
                }
                result(crendentials)
                break
            case "setHttpAuthCredential":
                let host = arguments!["host"] as! String
                let urlProtocol = arguments!["protocol"] as? String
                let urlPort = arguments!["port"] as? Int ?? 0
                var realm = arguments!["realm"] as? String;
                if let r = realm, r.isEmpty {
                    realm = nil
                }
                let username = arguments!["username"] as! String
                let password = arguments!["password"] as! String
                let credential = URLCredential(user: username, password: password, persistence: .permanent)
                CredentialDatabase.credentialStore.set(credential,
                                    for: URLProtectionSpace(host: host, port: urlPort, protocol: urlProtocol,
                                                            realm: realm, authenticationMethod: NSURLAuthenticationMethodHTTPBasic))
                result(true)
                break
            case "removeHttpAuthCredential":
                let host = arguments!["host"] as! String
                let urlProtocol = arguments!["protocol"] as? String
                let urlPort = arguments!["port"] as? Int ?? 0
                var realm = arguments!["realm"] as? String;
                if let r = realm, r.isEmpty {
                    realm = nil
                }
                let username = arguments!["username"] as! String
                let password = arguments!["password"] as! String
                
                var credential: URLCredential? = nil;
                var protectionSpaceCredential: URLProtectionSpace? = nil
                
                for (protectionSpace, credentials) in CredentialDatabase.credentialStore.allCredentials {
                    if protectionSpace.host == host && protectionSpace.realm == realm &&
                    protectionSpace.protocol == urlProtocol && protectionSpace.port == urlPort {
                        for c in credentials {
                            if c.value.user == username, c.value.password == password {
                                credential = c.value
                                protectionSpaceCredential = protectionSpace
                                break
                            }
                        }
                    }
                    if credential != nil {
                        break
                    }
                }
                
                if let c = credential, let protectionSpace = protectionSpaceCredential {
                    CredentialDatabase.credentialStore.remove(c, for: protectionSpace)
                }
                
                result(true)
                break
            case "removeHttpAuthCredentials":
                let host = arguments!["host"] as! String
                let urlProtocol = arguments!["protocol"] as? String
                let urlPort = arguments!["port"] as? Int ?? 0
                var realm = arguments!["realm"] as? String;
                if let r = realm, r.isEmpty {
                    realm = nil
                }
                
                var credentialsToRemove: [URLCredential] = [];
                var protectionSpaceCredential: URLProtectionSpace? = nil
                
                for (protectionSpace, credentials) in CredentialDatabase.credentialStore.allCredentials {
                    if protectionSpace.host == host && protectionSpace.realm == realm &&
                    protectionSpace.protocol == urlProtocol && protectionSpace.port == urlPort {
                        protectionSpaceCredential = protectionSpace
                        for c in credentials {
                            if let _ = c.value.user, let _ = c.value.password {
                                credentialsToRemove.append(c.value)
                            }
                        }
                        break
                    }
                }
                
                if let protectionSpace = protectionSpaceCredential {
                    for credential in credentialsToRemove {
                        CredentialDatabase.credentialStore.remove(credential, for: protectionSpace)
                    }
                }
                
                result(true)
                break
            case "clearAllAuthCredentials":
                for (protectionSpace, credentials) in CredentialDatabase.credentialStore.allCredentials {
                    for credential in credentials {
                        CredentialDatabase.credentialStore.remove(credential.value, for: protectionSpace)
                    }
                }
                result(true)
                break
            default:
                result(FlutterMethodNotImplemented)
                break
        }
    }
    
    public override func dispose() {
        super.dispose()
        plugin = nil
    }
    
    deinit {
        dispose()
    }
}
