//
//  PluginScript.swift
//  flutter_inappwebview
//
//  Created by Lorenzo Pichilli on 17/02/21.
//

import Foundation
import WebKit

public class PluginScript: UserScript {
    var requiredInAllContentWorlds = false
    var messageHandlerNames: [String] = []
    
    public override init(source: String, injectionTime: WKUserScriptInjectionTime, forMainFrameOnly: Bool) {
        super.init(source: source, injectionTime: injectionTime, forMainFrameOnly: forMainFrameOnly)
    }
    
    public init(groupName: String, source: String, injectionTime: WKUserScriptInjectionTime, forMainFrameOnly: Bool, requiredInAllContentWorlds: Bool = false, messageHandlerNames: [String] = []) {
        super.init(groupName: groupName, source: source, injectionTime: injectionTime, forMainFrameOnly: forMainFrameOnly)
        self.requiredInAllContentWorlds = requiredInAllContentWorlds
        self.messageHandlerNames = messageHandlerNames
    }
    
    @available(macOS 11.0, *)
    public override init(source: String, injectionTime: WKUserScriptInjectionTime, forMainFrameOnly: Bool, in contentWorld: WKContentWorld) {
        super.init(source: source, injectionTime: injectionTime, forMainFrameOnly: forMainFrameOnly, in: contentWorld)
        self.contentWorld = contentWorld
    }
    
    @available(macOS 11.0, *)
    public init(source: String, injectionTime: WKUserScriptInjectionTime, forMainFrameOnly: Bool, in contentWorld: WKContentWorld, requiredInAllContentWorlds: Bool = false, messageHandlerNames: [String] = []) {
        super.init(source: source, injectionTime: injectionTime, forMainFrameOnly: forMainFrameOnly, in: contentWorld)
        self.requiredInAllContentWorlds = requiredInAllContentWorlds
        self.messageHandlerNames = messageHandlerNames
    }

    @available(macOS 11.0, *)
    public init(groupName: String, source: String, injectionTime: WKUserScriptInjectionTime, forMainFrameOnly: Bool, in contentWorld: WKContentWorld, requiredInAllContentWorlds: Bool = false, messageHandlerNames: [String] = []) {
        super.init(groupName: groupName, source: source, injectionTime: injectionTime, forMainFrameOnly: forMainFrameOnly, in: contentWorld)
        self.requiredInAllContentWorlds = requiredInAllContentWorlds
        self.messageHandlerNames = messageHandlerNames
    }
    
    public func copyAndSet(groupName: String? = nil,
                           source: String? = nil,
                           injectionTime: WKUserScriptInjectionTime? = nil,
                           forMainFrameOnly: Bool? = nil,
                           requiredInAllContentWorlds: Bool? = nil,
                           messageHandlerNames: [String]? = nil) -> PluginScript {
        if #available(macOS 11.0, *) {
            return PluginScript(
                groupName: groupName ?? self.groupName!,
                source: source ?? self.source,
                injectionTime: injectionTime ?? self.injectionTime,
                forMainFrameOnly: forMainFrameOnly ?? self.isForMainFrameOnly,
                in: self.contentWorld,
                requiredInAllContentWorlds: requiredInAllContentWorlds ?? self.requiredInAllContentWorlds,
                messageHandlerNames: messageHandlerNames ?? self.messageHandlerNames
            )
        }
        return PluginScript(
            groupName: groupName ?? self.groupName!,
            source: source ?? self.source,
            injectionTime: injectionTime ?? self.injectionTime,
            forMainFrameOnly: forMainFrameOnly ?? self.isForMainFrameOnly,
            requiredInAllContentWorlds: requiredInAllContentWorlds ?? self.requiredInAllContentWorlds,
            messageHandlerNames: messageHandlerNames ?? self.messageHandlerNames
        )
    }
    
    @available(macOS 11.0, *)
    public func copyAndSet(groupName: String? = nil,
                           source: String? = nil,
                           injectionTime: WKUserScriptInjectionTime? = nil,
                           forMainFrameOnly: Bool? = nil,
                           contentWorld: WKContentWorld? = nil,
                           requiredInAllContentWorlds: Bool? = nil,
                           messageHandlerNames: [String]? = nil) -> PluginScript {
        return PluginScript(
            groupName: groupName ?? self.groupName!,
            source: source ?? self.source,
            injectionTime: injectionTime ?? self.injectionTime,
            forMainFrameOnly: forMainFrameOnly ?? self.isForMainFrameOnly,
            in: contentWorld ?? self.contentWorld,
            requiredInAllContentWorlds: requiredInAllContentWorlds ?? self.requiredInAllContentWorlds,
            messageHandlerNames: messageHandlerNames ?? self.messageHandlerNames
        )
    }
    
    static func == (lhs: PluginScript, rhs: PluginScript) -> Bool {
        if #available(macOS 11.0, *) {
            return lhs.groupName == rhs.groupName &&
                lhs.source == rhs.source &&
                lhs.injectionTime == rhs.injectionTime &&
                lhs.isForMainFrameOnly == rhs.isForMainFrameOnly &&
                lhs.contentWorld == rhs.contentWorld &&
                lhs.requiredInAllContentWorlds == rhs.requiredInAllContentWorlds &&
                lhs.messageHandlerNames == rhs.messageHandlerNames
        } else {
            return lhs.groupName == rhs.groupName &&
                lhs.source == rhs.source &&
                lhs.injectionTime == rhs.injectionTime &&
                lhs.isForMainFrameOnly == rhs.isForMainFrameOnly &&
                lhs.requiredInAllContentWorlds == rhs.requiredInAllContentWorlds &&
                lhs.messageHandlerNames == rhs.messageHandlerNames
        }
    }
}
