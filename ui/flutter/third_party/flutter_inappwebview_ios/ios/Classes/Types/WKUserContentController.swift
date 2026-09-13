//
//  UserContentController.swift
//  flutter_inappwebview
//
//  Created by Lorenzo Pichilli on 17/02/21.
//

import Foundation
import WebKit
import OrderedSet

extension WKUserContentController {
    static var WINDOW_ID_PREFIX = "WINDOW-ID-"

    // Workaround to create stored properties in an extension:
    // https://valv0.medium.com/computed-properties-and-extensions-a-pure-swift-approach-64733768112c

    @available(iOS 14.0, *)
    private static var _contentWorlds = [String: Set<WKContentWorld>]()
    @available(iOS 14.0, *)
    var contentWorlds: Set<WKContentWorld> {
        get {
            let tmpAddress = String(format: "%p", unsafeBitCast(self, to: Int.self))
            return WKUserContentController._contentWorlds[tmpAddress] ?? []
        }
        set(newValue) {
            let tmpAddress = String(format: "%p", unsafeBitCast(self, to: Int.self))
            WKUserContentController._contentWorlds[tmpAddress] = newValue
        }
    }

    private static var _userOnlyScripts = [String: [WKUserScriptInjectionTime:OrderedSet<UserScript>]]()
    var userOnlyScripts: [WKUserScriptInjectionTime:OrderedSet<UserScript>] {
        get {
            let tmpAddress = String(format: "%p", unsafeBitCast(self, to: Int.self))
            return WKUserContentController._userOnlyScripts[tmpAddress] ?? [:]
        }
        set(newValue) {
            let tmpAddress = String(format: "%p", unsafeBitCast(self, to: Int.self))
            WKUserContentController._userOnlyScripts[tmpAddress] = newValue
        }
    }

    private static var _pluginScripts = [String: [WKUserScriptInjectionTime:OrderedSet<PluginScript>]]()
    var pluginScripts: [WKUserScriptInjectionTime:OrderedSet<PluginScript>] {
        get {
            let tmpAddress = String(format: "%p", unsafeBitCast(self, to: Int.self))
            return WKUserContentController._pluginScripts[tmpAddress] ?? [:]
        }
        set(newValue) {
            let tmpAddress = String(format: "%p", unsafeBitCast(self, to: Int.self))
            WKUserContentController._pluginScripts[tmpAddress] = newValue
        }
    }

    public func initialize () {
        if #available(iOS 14.0, *) {
            contentWorlds = Set([WKContentWorld.page])
        }
        pluginScripts = [
            .atDocumentStart: OrderedSet(sequence: []),
            .atDocumentEnd: OrderedSet(sequence: []),
        ]
        userOnlyScripts = [
            .atDocumentStart: OrderedSet(sequence: []),
            .atDocumentEnd: OrderedSet(sequence: []),
        ]
    }

    public func dispose (windowId: Int64?) {
        if windowId == nil {
            let tmpAddress = String(format: "%p", unsafeBitCast(self, to: Int.self))
            if #available(iOS 14.0, *) {
                contentWorlds.removeAll()
                WKUserContentController._contentWorlds.removeValue(forKey: tmpAddress)
            }
            pluginScripts.removeAll()
            WKUserContentController._pluginScripts.removeValue(forKey: tmpAddress)
            userOnlyScripts.removeAll()
            WKUserContentController._userOnlyScripts.removeValue(forKey: tmpAddress)
        }
        else if #available(iOS 14.0, *), let windowId = windowId {
            let contentWorldsToRemove = contentWorlds.filter({ $0.windowId == windowId })
            for contentWorld in contentWorldsToRemove {
                contentWorlds.remove(contentWorld)
                removeAllScriptMessageHandlers(from: contentWorld)
            }
        }
    }

    public func sync(scriptMessageHandler: WKScriptMessageHandler) {
        let pluginScriptsList = pluginScripts.compactMap({ $0.value }).joined()
        for pluginScript in pluginScriptsList {
            if !containsPluginScript(pluginScript: pluginScript) {
                addUserScript(pluginScript)
                for messageHandlerName in pluginScript.messageHandlerNames {
                    removeScriptMessageHandler(forName: messageHandlerName)
                    add(scriptMessageHandler, name: messageHandlerName)
                }
            }
            if #available(iOS 14.0, *), pluginScript.requiredInAllContentWorlds {
                for contentWorld in contentWorlds {
                    if !containsPluginScript(pluginScript: pluginScript, in: contentWorld) {
                        let pluginScriptWithContentWorld = pluginScript.copyAndSet(contentWorld: contentWorld)
                        addUserScript(pluginScriptWithContentWorld)
                        for messageHandlerName in pluginScriptWithContentWorld.messageHandlerNames {
                            removeScriptMessageHandler(forName: messageHandlerName, contentWorld: contentWorld)
                            add(scriptMessageHandler, contentWorld: contentWorld, name: messageHandlerName)
                        }
                    }
                }
            }
        }

        let userOnlyScriptsList = userOnlyScripts.compactMap({ $0.value }).joined()
        for userOnlyScript in userOnlyScriptsList {
            if !userScripts.contains(userOnlyScript) {
                addUserScript(userOnlyScript)
            }
        }
    }

    public func addUserOnlyScript(_ userOnlyScript: UserScript) {
        if #available(iOS 14.0, *) {
            contentWorlds.insert(userOnlyScript.contentWorld)
        }
        userOnlyScripts[userOnlyScript.injectionTime]!.append(userOnlyScript)
    }

    public func addUserOnlyScripts(_ userOnlyScripts: [UserScript]) {
        for userOnlyScript in userOnlyScripts {
            addUserOnlyScript(userOnlyScript)
        }
    }

    public func addPluginScript(_ pluginScript: PluginScript) {
        if #available(iOS 14.0, *) {
            contentWorlds.insert(pluginScript.contentWorld)
        }
        pluginScripts[pluginScript.injectionTime]!.append(pluginScript)
    }

    public func addPluginScripts(_ pluginScripts: [PluginScript]) {
        for pluginScript in pluginScripts {
            addPluginScript(pluginScript)
        }
    }

    public func getPluginScriptsRequiredInAllContentWorlds() -> [PluginScript] {
        return pluginScripts.compactMap({ $0.value })
            .joined()
            .filter({ $0.injectionTime == .atDocumentStart && $0.requiredInAllContentWorlds })
    }

    @available(iOS 14.0, *)
    public func generateCodeForScriptEvaluation(scriptMessageHandler: WKScriptMessageHandler, source: String, contentWorld: WKContentWorld) -> String {
        let (inserted, _) = contentWorlds.insert(contentWorld)
        if inserted {
            var generatedCode = ""
            let pluginScriptsRequired = getPluginScriptsRequiredInAllContentWorlds()
            for pluginScript in pluginScriptsRequired {
                generatedCode += pluginScript.source + "\n"
                for messageHandlerName in pluginScript.messageHandlerNames {
                    removeScriptMessageHandler(forName: messageHandlerName, contentWorld: contentWorld)
                    add(scriptMessageHandler, contentWorld: contentWorld, name: messageHandlerName)
                }
            }
            if let windowId = contentWorld.windowId {
                generatedCode += "\(WINDOW_ID_VARIABLE_JS_SOURCE) = \(String(windowId));\n"
            }
            return generatedCode + "\n" + source
        }
        return source
    }

    public func removeUserOnlyScript(_ userOnlyScript: UserScript) {
        userOnlyScripts[userOnlyScript.injectionTime]!.remove(userOnlyScript)
        removeUserScript(scriptToRemove: userOnlyScript)
    }

    public func removeUserOnlyScript(at index: Int, injectionTime: WKUserScriptInjectionTime) {
        let scriptToRemove = userOnlyScripts[injectionTime]![index]
        userOnlyScripts[injectionTime]!.removeObject(at: index)
        removeUserScript(scriptToRemove: scriptToRemove)
    }

    public func removeAllUserOnlyScripts() {
        let allUserOnlyScripts = Array(userOnlyScripts.compactMap({ $0.value }).joined())

        userOnlyScripts[.atDocumentStart]!.removeAllObjects()
        userOnlyScripts[.atDocumentEnd]!.removeAllObjects()

        removeUserScripts(scriptsToRemove: allUserOnlyScripts)
    }

    public func removePluginScript(_ pluginScript: PluginScript) {
        pluginScripts[pluginScript.injectionTime]!.remove(pluginScript)
        for messageHandlerName in pluginScript.messageHandlerNames {
            removeScriptMessageHandler(forName: messageHandlerName)
            if #available(iOS 14.0, *) {
                for contentWorld in contentWorlds {
                    removeScriptMessageHandler(forName: messageHandlerName, contentWorld: contentWorld)
                }
            }
        }
        removeUserScript(scriptToRemove: pluginScript)
    }

    public func removeAllPluginScripts() {
        let allPluginScripts = Array(pluginScripts.compactMap({ $0.value }).joined())

        pluginScripts[.atDocumentStart]!.removeAllObjects()
        pluginScripts[.atDocumentEnd]!.removeAllObjects()

        removeUserScripts(scriptsToRemove: allPluginScripts)
    }

    public func removeAllPluginScriptMessageHandlers() {
        let allPluginScripts = pluginScripts.compactMap({ $0.value }).joined()
        for pluginScript in allPluginScripts {
            for messageHandlerName in pluginScript.messageHandlerNames {
                removeScriptMessageHandler(forName: messageHandlerName)
            }
        }
        if #available(iOS 14.0, *) {
            removeAllScriptMessageHandlers()
            for contentWorld in contentWorlds {
                removeAllScriptMessageHandlers(from: contentWorld)
            }
        }
    }

    @available(iOS 14.0, *)
    public func resetContentWorlds(windowId: Int64?) {
        let allUserOnlyScripts = userOnlyScripts.compactMap({ $0.value }).joined()
        let contentWorldsFiltered = contentWorlds.filter({ $0.windowId == windowId && $0 != WKContentWorld.page })
        for contentWorld in contentWorldsFiltered {
            var found = false
            for script in allUserOnlyScripts {
                if script.contentWorld == contentWorld {
                    found = true
                    break
                }
            }
            if !found {
                contentWorlds.remove(contentWorld)
            }
        }
    }

    private func removeUserScript(scriptToRemove: WKUserScript, shouldAddPreviousScripts: Bool = true) -> Void {
        // there isn't a way to remove a specific user script using WKUserContentController,
        // so we remove all the user scripts and, then, we add them again without the one that has been removed
        let userScripts = useCopyOfUserScripts()

        var userScriptsUpdated: [WKUserScript] = []
        for script in userScripts {
            if script != scriptToRemove {
                userScriptsUpdated.append(script)
            }
        }

        removeAllUserScripts()

        if shouldAddPreviousScripts {
            for script in userScriptsUpdated {
                addUserScript(script)
            }
        }
    }

    private func removeUserScripts(scriptsToRemove: [WKUserScript], shouldAddPreviousScripts: Bool = true) -> Void {
        // there isn't a way to remove a specific user script using WKUserContentController,
        // so we remove all the user scripts and, then, we add them again without the one that has been removed
        let userScripts = useCopyOfUserScripts()

        var userScriptsUpdated: [WKUserScript] = []
        for script in userScripts {
            if !scriptsToRemove.contains(script) {
                userScriptsUpdated.append(script)
            }
        }

        removeAllUserScripts()

        if shouldAddPreviousScripts {
            for script in userScriptsUpdated {
                addUserScript(script)
            }
        }
    }

    public func removeUserOnlyScripts(with groupName: String, shouldAddPreviousScripts: Bool = true) -> Void {
        let allUserOnlyScripts = userOnlyScripts.compactMap({ $0.value }).joined()
        var scriptsToRemove: [UserScript] = []
        for script in allUserOnlyScripts {
            if let scriptName = script.groupName, scriptName == groupName {
                scriptsToRemove.append(script)
                userOnlyScripts[script.injectionTime]!.remove(script)
            }
        }
        removeUserScripts(scriptsToRemove: scriptsToRemove, shouldAddPreviousScripts: shouldAddPreviousScripts)
    }

    public func removePluginScripts(with groupName: String, shouldAddPreviousScripts: Bool = true) -> Void {
        let allPluginScripts = pluginScripts.compactMap({ $0.value }).joined()
        var scriptsToRemove: [PluginScript] = []
        for script in allPluginScripts {
            if let scriptName = script.groupName, scriptName == groupName {
                scriptsToRemove.append(script)
                pluginScripts[script.injectionTime]!.remove(script)
            }
        }
        removeUserScripts(scriptsToRemove: scriptsToRemove, shouldAddPreviousScripts: shouldAddPreviousScripts)
    }

    public func containsPluginScript(pluginScript: PluginScript) -> Bool {
        let userScripts = useCopyOfUserScripts()
        for script in userScripts {
            if let script = script as? PluginScript, script == pluginScript {
                return true
            }
        }
        return false
    }
    
    public func containsPluginScript(with groupName: String) -> Bool {
        let userScripts = useCopyOfUserScripts()
        for script in userScripts {
            if let script = script as? PluginScript, script.groupName == groupName {
                return true
            }
        }
        return false
    }

    @available(iOS 14.0, *)
    public func containsPluginScript(pluginScript: PluginScript, in contentWorld: WKContentWorld) -> Bool {
        let userScripts = useCopyOfUserScripts()
        for script in userScripts {
            if let script = script as? PluginScript, script == pluginScript, script.contentWorld == contentWorld {
                return true
            }
        }
        return false
    }
    
    @available(iOS 14.0, *)
    public func containsPluginScript(with groupName: String, in contentWorld: WKContentWorld) -> Bool {
        let userScripts = useCopyOfUserScripts()
        for script in userScripts {
            if let script = script as? PluginScript, script.groupName == groupName, script.contentWorld == contentWorld {
                return true
            }
        }
        return false
    }

    @available(iOS 14.0, *)
    public func getContentWorlds(with windowId: Int64?) -> Set<WKContentWorld> {
        var contentWorldsFiltered = Set([WKContentWorld.page])
        let contentWorlds = Array(self.contentWorlds)
        for contentWorld in contentWorlds {
            if contentWorld.windowId == windowId {
                contentWorldsFiltered.insert(contentWorld)
            }
        }
        return contentWorldsFiltered
    }

    // use a copy of self.userScripts to avoid EXC_BREAKPOINT at runtime if self.userScripts gets removed when another code is looping them
    private func useCopyOfUserScripts() -> [WKUserScript] {
        return Array(self.userScripts)
    }
}
