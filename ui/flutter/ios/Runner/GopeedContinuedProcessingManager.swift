import BackgroundTasks
import Foundation
import Libgopeed
import UIKit

@available(iOS 26.0, *)
@objcMembers
final class GopeedContinuedProcessingManager: NSObject {

    static let shared = GopeedContinuedProcessingManager()

    // MARK: - Worker / state synchronization

    private let workerQueue = DispatchQueue(
        label: "com.gopeed.continued-processing",
        qos: .userInitiated
    )

    private let workerQueueKey = DispatchSpecificKey<Void>()

    private let stateLock = NSLock()
    private var enabledSnapshot = false
    private var handledTaskIDs = Set<String>()

    // MARK: - BGCPT state

    private var registeredIdentifiers = Set<String>()
    private var pendingTaskIDs = Set<String>()
    private var expiringTaskIDs = Set<String>()

    private var activeTasks:
        [String: BGContinuedProcessingTask] = [:]

    private var taskIdentifiers:
        [String: String] = [:]

    private var taskNames:
        [String: String] = [:]

    private var lastProgressUpdate:
        [String: Date] = [:]

    private var lastTitleUpdate:
        [String: Date] = [:]

    private let minimumProgressUpdateInterval:
        TimeInterval = 1.0

    private let minimumTitleUpdateInterval:
        TimeInterval = 1.0

    private override init() {
        super.init()

        workerQueue.setSpecific(
            key: workerQueueKey,
            value: ()
        )
    }


    // MARK: - Synchronization helpers

    private func syncOnWorker<T>(
        _ work: () -> T
    ) -> T {

        if DispatchQueue.getSpecific(
            key: workerQueueKey
        ) != nil {
            return work()
        }

        return workerQueue.sync(
            execute: work
        )
    }

    private func currentEnabledSnapshot() -> Bool {
        stateLock.lock()
        let value = enabledSnapshot
        stateLock.unlock()

        return value
    }

    private func setEnabledSnapshot(
        _ enabled: Bool
    ) {
        stateLock.lock()
        enabledSnapshot = enabled
        stateLock.unlock()
    }

    private func setHandledSnapshot(
        taskID: String,
        handled: Bool
    ) {
        stateLock.lock()

        if handled {
            handledTaskIDs.insert(taskID)
        } else {
            handledTaskIDs.remove(taskID)
        }

        stateLock.unlock()
    }

    private func handledTaskIDsSnapshot() -> [String] {
        stateLock.lock()
        let ids = Array(handledTaskIDs)
        stateLock.unlock()

        return ids
    }

    private func setTaskOwnership(
        taskID: String,
        handled: Bool
    ) {
        setHandledSnapshot(
            taskID: taskID,
            handled: handled
        )

        // This is a second layer of protection in addition to the
        // Objective-C forwarder. If Continued Processing owns a task,
        // the custom ActivityKit manager must not create a second
        // Live Activity for it.
        GopeedLiveActivityManager.shared
            .setTaskSuppressed(
                handled,
                taskID: taskID
            )
    }


    // MARK: - Setting

    func setEnabled(
        _ enabled: Bool
    ) -> Bool {

        return syncOnWorker {

            self.setEnabledSnapshot(enabled)

            if !enabled {
                self.stopAllContinuedTasks()
            }

            print(
                "ContinuedProcessing: enabled =",
                enabled
            )

            return true
        }
    }


    // MARK: - Event entry point

    func handleTaskEventPayload(
        _ payload: String
    ) {

        guard currentEnabledSnapshot() else {
            return
        }

        // Pre-mark task.start so the Objective-C forwarder can
        // immediately suppress the custom ActivityKit Live Activity.
        if
            let data = payload.data(using: .utf8),
            let json = try? JSONSerialization.jsonObject(
                with: data
            ) as? [String: Any],
            let type = json["type"] as? String,
            type == "task.start",
            let taskID = json["taskId"] as? String
        {
            setTaskOwnership(
                taskID: taskID,
                handled: true
            )
        }

        // IMPORTANT: asynchronous, so Gopeed's event thread
        // is never blocked by BGCPT bookkeeping.
        workerQueue.async { [weak self] in
            self?.handleTaskEventPayloadOnWorker(
                payload
            )
        }
    }

    private func handleTaskEventPayloadOnWorker(
        _ payload: String
    ) {

        guard currentEnabledSnapshot() else {
            return
        }

        guard
            let data = payload.data(
                using: .utf8
            ),
            let json =
                try? JSONSerialization
                    .jsonObject(
                        with: data
                    ) as? [String: Any],
            let type =
                json["type"] as? String,
            let taskID =
                json["taskId"] as? String
        else {
            print(
                "ContinuedProcessing: invalid event"
            )
            return
        }

        let name =
            json["name"] as? String
            ?? "Download"

        switch type {

        case "task.start":
            beginTask(
                taskID: taskID,
                name: name
            )

        case "task.progress":
            updateProgress(
                taskID: taskID,
                force: false
            )

        case "task.pause":
            finishTask(
                taskID: taskID,
                success: true,
                finalSubtitle: "Paused"
            )

        case "task.done":
            finishTask(
                taskID: taskID,
                success: true,
                finalSubtitle:
                    "Download complete"
            )

        case "task.error":
            finishTask(
                taskID: taskID,
                success: false,
                finalSubtitle:
                    "Download failed"
            )

        case "task.delete":
            finishTask(
                taskID: taskID,
                success: true,
                finalSubtitle:
                    "Download removed"
            )

        default:
            break
        }
    }


    // MARK: - Used by Objective-C forwarder

    @objc(isHandlingTaskId:)
    func isHandlingTaskId(
        _ taskID: String
    ) -> Bool {

        // Do not wait for workerQueue here. The forwarder calls
        // this for frequent progress events.
        stateLock.lock()
        let handled = handledTaskIDs.contains(taskID)
        stateLock.unlock()

        return handled
    }


    // MARK: - Start BGCPT

    private func beginTask(
        taskID: String,
        name: String
    ) {

        guard currentEnabledSnapshot() else {
            setTaskOwnership(
                taskID: taskID,
                handled: false
            )
            return
        }

        // BGCPT should originate from a foreground user action.
        // This main-thread check happens only once per task start.
        let appIsActive: Bool

        if Thread.isMainThread {

            appIsActive =
                UIApplication.shared
                    .applicationState == .active

        } else {

            appIsActive =
                DispatchQueue.main.sync {
                    UIApplication.shared
                        .applicationState == .active
                }
        }

        guard appIsActive else {

            print(
                "ContinuedProcessing:",
                "ignored non-foreground start",
                taskID
            )

            setTaskOwnership(
                taskID: taskID,
                handled: false
            )

            return
        }

        guard
            !pendingTaskIDs.contains(taskID),
            activeTasks[taskID] == nil
        else {
            setTaskOwnership(
                taskID: taskID,
                handled: true
            )
            return
        }

        let identifier =
            makeIdentifier(
                taskID: taskID
            )

        taskIdentifiers[taskID] =
            identifier

        taskNames[taskID] =
            name

        if !registeredIdentifiers
            .contains(identifier) {

            let registered =
                BGTaskScheduler.shared.register(
                    forTaskWithIdentifier:
                        identifier,
                    using: nil
                ) { [weak self] task in

                    guard
                        let self,
                        let continuedTask =
                            task as?
                            BGContinuedProcessingTask
                    else {
                        task.setTaskCompleted(
                            success: false
                        )
                        return
                    }

                    self.workerQueue.async {

                        self.activateTask(
                            continuedTask,
                            taskID: taskID,
                            name: name
                        )
                    }
                }

            guard registered else {

                print(
                    "ContinuedProcessing:",
                    "registration failed:",
                    identifier
                )

                taskIdentifiers.removeValue(
                    forKey: taskID
                )

                taskNames.removeValue(
                    forKey: taskID
                )

                setTaskOwnership(
                    taskID: taskID,
                    handled: false
                )

                return
            }

            registeredIdentifiers.insert(
                identifier
            )
        }

        let request =
            BGContinuedProcessingTaskRequest(
                identifier: identifier,
                title: name,
                subtitle:
                    "Preparing download…"
            )

        request.strategy = .fail

        pendingTaskIDs.insert(taskID)

        setTaskOwnership(
            taskID: taskID,
            handled: true
        )

        do {

            try BGTaskScheduler.shared.submit(
                request
            )

            print(
                "ContinuedProcessing:",
                "submitted:",
                identifier
            )

        } catch {

            pendingTaskIDs.remove(
                taskID
            )

            taskIdentifiers.removeValue(
                forKey: taskID
            )

            taskNames.removeValue(
                forKey: taskID
            )

            setTaskOwnership(
                taskID: taskID,
                handled: false
            )

            print(
                "ContinuedProcessing:",
                "submission failed:",
                error
            )
        }
    }


    // MARK: - System launched task

    private func activateTask(
        _ task: BGContinuedProcessingTask,
        taskID: String,
        name: String
    ) {

        guard currentEnabledSnapshot() else {

            setTaskOwnership(
                taskID: taskID,
                handled: false
            )

            task.setTaskCompleted(
                success: false
            )

            return
        }

        pendingTaskIDs.remove(taskID)

        activeTasks[taskID] = task
        taskNames[taskID] = name

        setTaskOwnership(
            taskID: taskID,
            handled: true
        )

        task.progress.totalUnitCount = 100
        task.progress.completedUnitCount = 0

        task.expirationHandler = {
            [weak self, weak task] in

            guard
                let self,
                let task
            else {
                return
            }

            self.workerQueue.async {

                self.handleExpiration(
                    task,
                    taskID: taskID
                )
            }
        }

        print(
            "ContinuedProcessing:",
            "started:",
            taskID
        )

        updateProgress(
            taskID: taskID,
            force: true
        )
    }


    // MARK: - Async Gopeed invocation

    private func invokeGopeed(
        method: String,
        path: String,
        query: String = "",
        body: String = "",
        completion:
            @escaping (String?) -> Void
    ) {

        GopeedInvokeAsyncNative(
            method,
            path,
            query,
            body
        ) { [weak self] success, payload in

            guard let self else {
                return
            }

            self.workerQueue.async {
                completion(
                    success
                    ? payload
                    : nil
                )
            }
        }
    }


    // MARK: - Progress

    private struct RuntimeStatus {
        let status: String
        let downloaded: Int64
        let total: Int64
        let speed: Int64
    }

    private func getRuntimeStatus(
        taskID: String,
        completion:
            @escaping (RuntimeStatus?) -> Void
    ) {

        invokeGopeed(
            method: "GET",
            path:
                "/api/v1/tasks/\(taskID)/status"
        ) { response in

            guard
                let response,
                let data =
                    response.data(
                        using: .utf8
                    ),
                let root =
                    try? JSONSerialization
                        .jsonObject(
                            with: data
                        ) as? [String: Any],
                let code =
                    (root["code"] as? NSNumber)?
                        .intValue,
                code == 0,
                let body =
                    root["data"]
                        as? [String: Any]
            else {
                completion(nil)
                return
            }

            completion(
                RuntimeStatus(
                    status:
                        body["status"]
                            as? String ?? "",
                    downloaded:
                        (body["downloaded"]
                            as? NSNumber)?
                            .int64Value ?? 0,
                    total:
                        (body["total"]
                            as? NSNumber)?
                            .int64Value ?? 0,
                    speed:
                        (body["speed"]
                            as? NSNumber)?
                            .int64Value ?? 0
                )
            )
        }
    }

    private func updateProgress(
        taskID: String,
        force: Bool
    ) {

        guard activeTasks[taskID] != nil else {
            return
        }

        let requestTime = Date()

        if !force,
           let previous =
                lastProgressUpdate[taskID],
           requestTime.timeIntervalSince(previous)
                < minimumProgressUpdateInterval {

            return
        }

        // Mark before starting the async request so frequent
        // task.progress events cannot create overlapping
        // status requests.
        lastProgressUpdate[taskID] =
            requestTime

        getRuntimeStatus(
            taskID: taskID
        ) { [weak self] runtime in

            guard
                let self,
                let runtime,
                let task =
                    self.activeTasks[taskID]
            else {
                return
            }

            let now = Date()

            let shouldUpdateTitle: Bool

            if force {

                shouldUpdateTitle = true

            } else if let previous =
                        self.lastTitleUpdate[
                            taskID
                        ] {

                shouldUpdateTitle =
                    now.timeIntervalSince(
                        previous
                    )
                    >=
                    self.minimumTitleUpdateInterval

            } else {

                shouldUpdateTitle = true
            }

            let downloaded =
                max(
                    runtime.downloaded,
                    0
                )

            let name =
                self.taskNames[taskID]
                ?? "Download"

            if runtime.total > 0 {

                let total =
                    max(runtime.total, 1)

                let completed =
                    min(
                        downloaded,
                        total
                    )

                task.progress
                    .totalUnitCount =
                    total

                task.progress
                    .completedUnitCount =
                    completed

                if shouldUpdateTitle {

                    let percent =
                        Int(
                            (
                                Double(
                                    completed
                                )
                                / Double(total)
                                * 100.0
                            ).rounded()
                        )

                    var subtitle =
                        "\(percent)% • " +
                        "\(self.formatBytes(completed)) / " +
                        "\(self.formatBytes(total))"

                    if runtime.speed > 0 {
                        subtitle +=
                            " • " +
                            "\(self.formatBytes(runtime.speed))/s"
                    }

                    task.updateTitle(
                        name,
                        subtitle:
                            subtitle
                    )

                    self.lastTitleUpdate[
                        taskID
                    ] = now
                }

            } else {

                task.progress
                    .totalUnitCount = 100

                task.progress
                    .completedUnitCount = 0

                if shouldUpdateTitle {

                    var subtitle =
                        self.formatBytes(
                            downloaded
                        )

                    if runtime.speed > 0 {
                        subtitle +=
                            " • " +
                            "\(self.formatBytes(runtime.speed))/s"
                    }

                    task.updateTitle(
                        name,
                        subtitle:
                            subtitle
                    )

                    self.lastTitleUpdate[
                        taskID
                    ] = now
                }
            }
        }
    }

    // MARK: - Completion

    private func finishTask(
        taskID: String,
        success: Bool,
        finalSubtitle: String
    ) {

        if let identifier =
            taskIdentifiers[taskID] {

            BGTaskScheduler.shared.cancel(
                taskRequestWithIdentifier:
                    identifier
            )
        }

        pendingTaskIDs.remove(taskID)

        if let task =
            activeTasks.removeValue(
                forKey: taskID
            ) {

            if success,
               finalSubtitle ==
                    "Download complete",
               task.progress
                    .totalUnitCount > 0 {

                task.progress
                    .completedUnitCount =
                    task.progress
                        .totalUnitCount
            }

            task.updateTitle(
                taskNames[taskID]
                    ?? "Download",
                subtitle:
                    finalSubtitle
            )

            task.setTaskCompleted(
                success: success
            )
        }

        taskIdentifiers.removeValue(
            forKey: taskID
        )

        taskNames.removeValue(
            forKey: taskID
        )

        lastProgressUpdate.removeValue(
            forKey: taskID
        )

        lastTitleUpdate.removeValue(
            forKey: taskID
        )

        setTaskOwnership(
            taskID: taskID,
            handled: false
        )

        print(
            "ContinuedProcessing:",
            "finished:",
            taskID,
            "success:",
            success
        )
    }

    // MARK: - Expiration / system cancellation

    private func handleExpiration(
        _ task: BGContinuedProcessingTask,
        taskID: String
    ) {

        activeTasks.removeValue(
            forKey: taskID
        )

        pendingTaskIDs.remove(taskID)
        expiringTaskIDs.insert(taskID)

        lastProgressUpdate.removeValue(
            forKey: taskID
        )

        lastTitleUpdate.removeValue(
            forKey: taskID
        )

        // Keep ownership/suppression active until the pause request
        // has been delivered. Otherwise progress events arriving in
        // this small window can start a custom ActivityKit activity.
        setTaskOwnership(
            taskID: taskID,
            handled: true
        )

        print(
            "ContinuedProcessing:",
            "expired/cancelled:",
            taskID
        )

        task.setTaskCompleted(
            success: false
        )

        invokeGopeed(
            method: "PUT",
            path:
                "/api/v1/tasks/\(taskID)/pause"
        ) { [weak self] _ in
            self?.finishExpirationCleanup(
                taskID: taskID
            )
        }

        // Defensive fallback in case the async bridge never calls
        // back because the process is being suspended.
        workerQueue.asyncAfter(
            deadline: .now() + 2.0
        ) { [weak self] in
            self?.finishExpirationCleanup(
                taskID: taskID
            )
        }
    }

    private func finishExpirationCleanup(
        taskID: String
    ) {

        guard expiringTaskIDs.remove(taskID) != nil else {
            return
        }

        taskIdentifiers.removeValue(
            forKey: taskID
        )

        taskNames.removeValue(
            forKey: taskID
        )

        setTaskOwnership(
            taskID: taskID,
            handled: false
        )
    }


    // MARK: - Disable all BGCPT tasks

    private func stopAllContinuedTasks() {

        let ownedTaskIDs =
            Set(
                handledTaskIDsSnapshot()
                + Array(taskIdentifiers.keys)
                + Array(pendingTaskIDs)
                + Array(activeTasks.keys)
                + Array(expiringTaskIDs)
            )

        for (
            taskID,
            identifier
        ) in taskIdentifiers {

            BGTaskScheduler.shared.cancel(
                taskRequestWithIdentifier:
                    identifier
            )

            if let task =
                activeTasks[taskID] {

                task.setTaskCompleted(
                    success: true
                )
            }
        }

        activeTasks.removeAll()
        pendingTaskIDs.removeAll()
        expiringTaskIDs.removeAll()
        taskIdentifiers.removeAll()
        taskNames.removeAll()
        lastProgressUpdate.removeAll()
        lastTitleUpdate.removeAll()

        for taskID in ownedTaskIDs {
            setTaskOwnership(
                taskID: taskID,
                handled: false
            )
        }
    }


    // MARK: - Helpers

    private func makeIdentifier(
        taskID: String
    ) -> String {

        let bundleID =
            Bundle.main.bundleIdentifier
            ?? "com.gopeed.gopeed"

        let safeID =
            taskID.replacingOccurrences(
                of: "[^A-Za-z0-9_-]",
                with: "-",
                options:
                    .regularExpression
            )

        return
            "\(bundleID)" +
            ".continuedDownload." +
            safeID
    }

    private func formatBytes(
        _ bytes: Int64
    ) -> String {

        return ByteCountFormatter.string(
            fromByteCount:
                max(bytes, 0),
            countStyle: .file
        )
    }
}
