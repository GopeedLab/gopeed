import ActivityKit
import Foundation
import Libgopeed
import UIKit

@objcMembers
final class GopeedLiveActivityManager: NSObject {

    static let shared = GopeedLiveActivityManager()

    // Gopeed emits progress approximately every 350 ms.
    // There is no reason to hit ActivityKit that frequently.
    private let minimumUpdateInterval: TimeInterval = 2.0

    private let stateLock = NSLock()

    private var lastUpdateTime: [String: Date] = [:]
    private var suppressedTaskIDs = Set<String>()
    private var activityCreationInProgress = Set<String>()

    private override init() {
        super.init()
    }


    // MARK: - BGCPT coordination

    func setTaskSuppressed(
        _ suppressed: Bool,
        taskID: String
    ) {
        stateLock.lock()

        if suppressed {
            suppressedTaskIDs.insert(taskID)
        } else {
            suppressedTaskIDs.remove(taskID)
        }

        stateLock.unlock()

        if suppressed {
            guard #available(iOS 16.2, *) else {
                return
            }

            Task { [weak self] in
                guard let self else {
                    return
                }

                // Suppression may have been removed again if BGCPT
                // submission failed before this async cleanup runs.
                guard self.isTaskSuppressed(taskID) else {
                    return
                }

                await self.removeAllActivities(
                    taskID: taskID,
                    reason: "suppressed by continued processing"
                )
            }
        }
    }

    private func isTaskSuppressed(
        _ taskID: String
    ) -> Bool {
        stateLock.lock()
        let suppressed =
            suppressedTaskIDs.contains(taskID)
        stateLock.unlock()

        return suppressed
    }

    private func beginActivityCreation(
        taskID: String
    ) -> Bool {
        stateLock.lock()
        defer { stateLock.unlock() }

        guard
            !suppressedTaskIDs.contains(taskID),
            !activityCreationInProgress.contains(taskID)
        else {
            return false
        }

        activityCreationInProgress.insert(taskID)
        return true
    }

    private func endActivityCreation(
        taskID: String
    ) {
        stateLock.lock()
        activityCreationInProgress.remove(taskID)
        stateLock.unlock()
    }

    private func shouldHandleProgress(
        taskID: String,
        now: Date
    ) -> Bool {
        stateLock.lock()
        defer { stateLock.unlock() }

        guard !suppressedTaskIDs.contains(taskID) else {
            return false
        }

        if let previous = lastUpdateTime[taskID],
           now.timeIntervalSince(previous)
                < minimumUpdateInterval {
            return false
        }

        lastUpdateTime[taskID] = now
        return true
    }

    private func clearUpdateTime(
        taskID: String
    ) {
        stateLock.lock()
        lastUpdateTime.removeValue(
            forKey: taskID
        )
        stateLock.unlock()
    }

    // MARK: - Gopeed event entry point

    @objc
    func handleTaskEventPayload(_ payload: String) {

        guard #available(iOS 16.2, *) else {
            return
        }

        guard
            let data = payload.data(using: .utf8),
            let json = try? JSONSerialization.jsonObject(with: data)
                as? [String: Any],
            let type = json["type"] as? String,
            let taskID = json["taskId"] as? String
        else {
            print("LiveActivity: invalid task event payload")
            return
        }

        guard !isTaskSuppressed(taskID) else {
            return
        }

        let name = json["name"] as? String ?? "Download"

        switch type {

        case "task.start":
            Task {
                await refreshActivity(
                    taskID: taskID,
                    name: name,
                    allowStart: true
                )
            }

        case "task.progress":
            handleProgress(
                taskID: taskID,
                name: name
            )

        case "task.pause":
            Task {
                await refreshActivity(
                    taskID: taskID,
                    name: name,
                    allowStart: false
                )
            }

        case "task.done":
            Task {
                await finishActivity(
                    taskID: taskID
                )
            }

        case "task.error":
            let error =
                json["error"] as? String
                ?? "Download failed"

            Task {
                await failActivity(
                    taskID: taskID,
                    error: error
                )
            }

        case "task.delete":
            Task {
                await removeActivity(
                    taskID: taskID
                )
            }

        default:
            break
        }
    }


    // MARK: - Progress throttling

    @available(iOS 16.2, *)
    private func handleProgress(
        taskID: String,
        name: String
    ) {
        let now = Date()

        guard shouldHandleProgress(
            taskID: taskID,
            now: now
        ) else {
            return
        }

        Task {
            await refreshActivity(
                taskID: taskID,
                name: name,
                allowStart: true
            )
        }
    }


    // MARK: - Async Gopeed invocation

    private func invokeGopeed(
        method: String,
        path: String,
        query: String = "",
        body: String = ""
    ) async -> String? {

        await withCheckedContinuation {
            continuation in

            GopeedInvokeAsyncNative(
                method,
                path,
                query,
                body
            ) { success, payload in

                continuation.resume(
                    returning:
                        success
                        ? payload
                        : nil
                )
            }
        }
    }


    // MARK: - Gopeed runtime status

    private struct RuntimeStatus {
        let status: String
        let downloaded: Int64
        let total: Int64
        let speed: Int64
    }

    private func getRuntimeStatus(
        taskID: String
    ) async -> RuntimeStatus? {

        guard
            let response =
                await invokeGopeed(
                    method: "GET",
                    path:
                        "/api/v1/tasks/\(taskID)/status"
                )
        else {
            print(
                "LiveActivity: status request failed for \(taskID)"
            )
            return nil
        }

        guard
            let data = response.data(using: .utf8),
            let root =
                try? JSONSerialization.jsonObject(
                    with: data
                ) as? [String: Any]
        else {
            print(
                "LiveActivity: failed to decode status for \(taskID)"
            )
            return nil
        }

        let code =
            (root["code"] as? NSNumber)?.intValue
            ?? -1

        guard code == 0 else {
            let message =
                root["msg"] as? String
                ?? "Unknown Gopeed API error"

            print(
                "LiveActivity: Gopeed status error:",
                message
            )

            return nil
        }

        guard
            let body =
                root["data"] as? [String: Any]
        else {
            return nil
        }

        return RuntimeStatus(
            status:
                body["status"] as? String ?? "",
            downloaded:
                (body["downloaded"] as? NSNumber)?
                    .int64Value ?? 0,
            total:
                (body["total"] as? NSNumber)?
                    .int64Value ?? 0,
            speed:
                (body["speed"] as? NSNumber)?
                    .int64Value ?? 0
        )
    }

    // MARK: - Build Live Activity state

    @available(iOS 16.2, *)
    private func makeState(
        from runtime: RuntimeStatus
    ) -> GopeedDownloadAttributes.ContentState {

        let now = Date()

        let total =
            max(runtime.total, 0)

        let downloaded: Int64

        if total > 0 {
            downloaded =
                min(
                    max(runtime.downloaded, 0),
                    total
                )
        } else {
            downloaded =
                max(runtime.downloaded, 0)
        }

        let progress: Double

        if total > 0 {
            progress =
                min(
                    max(
                        Double(downloaded)
                            / Double(total),
                        0
                    ),
                    1
                )
        } else {
            progress = 0
        }

        // If we don't have enough information for an ETA,
        // use the real static progress bar.
        guard
            runtime.status == "running",
            total > 0,
            runtime.speed > 0,
            downloaded < total,
            progress > 0,
            progress < 1
        else {
            return GopeedDownloadAttributes
                .ContentState(
                    progress: progress,
                    downloaded: downloaded,
                    total: total,
                    speed: runtime.speed,
                    status: runtime.status,
                    estimatedStart: now,
                    estimatedEnd:
                        now.addingTimeInterval(1),
                    usesEstimatedProgress: false
                )
        }

        let remainingBytes =
            Double(total - downloaded)

        var remainingSeconds =
            remainingBytes
            / Double(runtime.speed)

        // Don't allow corrupt speed values to create
        // ridiculous date ranges.
        remainingSeconds =
            min(
                max(remainingSeconds, 1),
                86_400
            )

        let remainingFraction =
            max(1.0 - progress, 0.001)

        let estimatedTotalDuration =
            remainingSeconds
            / remainingFraction

        let elapsedEstimate =
            estimatedTotalDuration
            * progress

        let estimatedStart =
            now.addingTimeInterval(
                -elapsedEstimate
            )

        let estimatedEnd =
            now.addingTimeInterval(
                remainingSeconds
            )

        return GopeedDownloadAttributes
            .ContentState(
                progress: progress,
                downloaded: downloaded,
                total: total,
                speed: runtime.speed,
                status: runtime.status,
                estimatedStart: estimatedStart,
                estimatedEnd: estimatedEnd,
                usesEstimatedProgress: true
            )
    }


    // MARK: - Find existing Activity

    @available(iOS 16.2, *)
    private func findActivities(
        taskID: String
    ) -> [Activity<GopeedDownloadAttributes>] {

        return Activity<
            GopeedDownloadAttributes
        >
        .activities
        .filter {
            $0.attributes.taskId == taskID
        }
    }

    @available(iOS 16.2, *)
    private func collapseDuplicateActivities(
        _ activities: [Activity<GopeedDownloadAttributes>],
        content: ActivityContent<GopeedDownloadAttributes.ContentState>
    ) async {

        guard let primary = activities.first else {
            return
        }

        await primary.update(content)

        if activities.count > 1 {
            for duplicate in activities.dropFirst() {
                await duplicate.end(
                    nil,
                    dismissalPolicy: .immediate
                )
            }

            print(
                "LiveActivity: removed",
                activities.count - 1,
                "duplicate activity/activities for",
                primary.attributes.taskId
            )
        }
    }


    // MARK: - Start / update Activity

    @available(iOS 16.2, *)
    private func refreshActivity(
        taskID: String,
        name: String,
        allowStart: Bool
    ) async {

        guard !isTaskSuppressed(taskID) else {
            return
        }

        guard
            let runtime =
                await getRuntimeStatus(
                    taskID: taskID
                )
        else {
            return
        }

        guard !isTaskSuppressed(taskID) else {
            return
        }

        let state =
            makeState(from: runtime)

        let content =
            ActivityContent(
                state: state,
                staleDate: nil
            )

        let existing =
            findActivities(
                taskID: taskID
            )

        if !existing.isEmpty {
            await collapseDuplicateActivities(
                existing,
                content: content
            )
            return
        }

        guard allowStart else {
            return
        }

        guard
            ActivityAuthorizationInfo()
                .areActivitiesEnabled
        else {
            print(
                "LiveActivity: activities disabled"
            )
            return
        }

        let appIsActive =
            await MainActor.run {
                UIApplication.shared
                    .applicationState == .active
            }

        guard appIsActive else {
            return
        }

        // Only one async caller may pass the creation gate for
        // a given Gopeed task. This prevents task.start and early
        // task.progress callbacks from creating multiple activities
        // while their async status requests overlap.
        guard beginActivityCreation(
            taskID: taskID
        ) else {
            return
        }

        defer {
            endActivityCreation(
                taskID: taskID
            )
        }

        guard !isTaskSuppressed(taskID) else {
            return
        }

        // Re-check after acquiring the creation gate because an
        // earlier request may have created the activity already.
        let recheck =
            findActivities(
                taskID: taskID
            )

        if !recheck.isEmpty {
            await collapseDuplicateActivities(
                recheck,
                content: content
            )
            return
        }

        let attributes =
            GopeedDownloadAttributes(
                taskId: taskID,
                fileName: name
            )

        do {
            let activity =
                try Activity.request(
                    attributes: attributes,
                    content: content,
                    pushType: nil
                )

            // Continued Processing may have claimed the task while
            // Activity.request was in progress. If so, remove this
            // custom activity immediately.
            if isTaskSuppressed(taskID) {
                await activity.end(
                    nil,
                    dismissalPolicy: .immediate
                )
                return
            }

            print(
                "LiveActivity: started",
                activity.id,
                taskID
            )

        } catch {
            print(
                "LiveActivity: start failed:",
                error
            )
        }
    }


    // MARK: - Complete Activity

    @available(iOS 16.2, *)
    private func finishActivity(
        taskID: String
    ) async {

        let activities =
            findActivities(
                taskID: taskID
            )

        guard !activities.isEmpty else {
            clearUpdateTime(
                taskID: taskID
            )
            return
        }

        let now = Date()

        for activity in activities {
            let oldState =
                activity.content.state

            let finalTotal =
                max(
                    oldState.total,
                    oldState.downloaded
                )

            let finalState =
                GopeedDownloadAttributes
                    .ContentState(
                        progress: 1.0,
                        downloaded: finalTotal,
                        total: finalTotal,
                        speed: 0,
                        status: "done",
                        estimatedStart: now,
                        estimatedEnd:
                            now.addingTimeInterval(1),
                        usesEstimatedProgress: false
                    )

            let finalContent =
                ActivityContent(
                    state: finalState,
                    staleDate: nil
                )

            await activity.end(
                finalContent,
                dismissalPolicy:
                    .after(
                        Date()
                            .addingTimeInterval(15)
                    )
            )
        }

        clearUpdateTime(
            taskID: taskID
        )

        print(
            "LiveActivity: completed",
            taskID
        )
    }


    // MARK: - Error Activity

    @available(iOS 16.2, *)
    private func failActivity(
        taskID: String,
        error: String
    ) async {

        let activities =
            findActivities(
                taskID: taskID
            )

        guard !activities.isEmpty else {
            clearUpdateTime(
                taskID: taskID
            )
            return
        }

        let now = Date()

        for activity in activities {
            let old =
                activity.content.state

            let failedState =
                GopeedDownloadAttributes
                    .ContentState(
                        progress: old.progress,
                        downloaded: old.downloaded,
                        total: old.total,
                        speed: 0,
                        status: "error",
                        estimatedStart: now,
                        estimatedEnd:
                            now.addingTimeInterval(1),
                        usesEstimatedProgress: false
                    )

            let content =
                ActivityContent(
                    state: failedState,
                    staleDate: nil
                )

            await activity.end(
                content,
                dismissalPolicy:
                    .after(
                        Date()
                            .addingTimeInterval(30)
                    )
            )
        }

        clearUpdateTime(
            taskID: taskID
        )

        print(
            "LiveActivity: task failed:",
            taskID,
            error
        )
    }


    // MARK: - Delete / suppression cleanup

    @available(iOS 16.2, *)
    private func removeActivity(
        taskID: String
    ) async {

        await removeAllActivities(
            taskID: taskID,
            reason: "removed"
        )
    }

    @available(iOS 16.2, *)
    private func removeAllActivities(
        taskID: String,
        reason: String
    ) async {

        let activities =
            findActivities(
                taskID: taskID
            )

        for activity in activities {
            await activity.end(
                nil,
                dismissalPolicy: .immediate
            )
        }

        clearUpdateTime(
            taskID: taskID
        )

        if !activities.isEmpty {
            print(
                "LiveActivity:",
                reason,
                taskID,
                "count:",
                activities.count
            )
        }
    }

}
