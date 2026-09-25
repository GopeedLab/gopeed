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
    private var taskGenerations: [String: UUID] = [:]
    private var activityCreationInProgress:
        [String: UUID] = [:]

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
            taskGenerations.removeValue(
                forKey: taskID
            )
            lastUpdateTime.removeValue(
                forKey: taskID
            )
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

    private func startGeneration(
        taskID: String
    ) -> UUID {
        let generation = UUID()

        stateLock.lock()
        taskGenerations[taskID] = generation
        lastUpdateTime.removeValue(
            forKey: taskID
        )
        stateLock.unlock()

        return generation
    }

    private func currentGeneration(
        taskID: String
    ) -> UUID? {
        stateLock.lock()
        let generation =
            taskGenerations[taskID]
        stateLock.unlock()

        return generation
    }

    private func generationForProgress(
        taskID: String
    ) -> UUID? {
        stateLock.lock()
        defer { stateLock.unlock() }

        guard
            !suppressedTaskIDs.contains(taskID)
        else {
            return nil
        }

        if let generation =
            taskGenerations[taskID] {
            return generation
        }

        // Progress may be the first event after Continued Processing
        // releases ownership. Establish a fresh lifecycle generation.
        // refreshActivity still verifies that Gopeed reports the task
        // as running before it can create a Live Activity.
        let generation = UUID()

        taskGenerations[taskID] =
            generation

        lastUpdateTime.removeValue(
            forKey: taskID
        )

        return generation
    }

    @discardableResult
    private func invalidateGeneration(
        taskID: String
    ) -> UUID? {
        stateLock.lock()
        let generation =
            taskGenerations.removeValue(
                forKey: taskID
            )
        lastUpdateTime.removeValue(
            forKey: taskID
        )
        stateLock.unlock()

        return generation
    }

    private func isCurrentGeneration(
        taskID: String,
        generation: UUID
    ) -> Bool {
        stateLock.lock()
        let isCurrent =
            taskGenerations[taskID]
            == generation
        stateLock.unlock()

        return isCurrent
    }

    private func beginActivityCreation(
        taskID: String,
        generation: UUID
    ) -> Bool {
        stateLock.lock()
        defer { stateLock.unlock() }

        guard
            !suppressedTaskIDs.contains(taskID),
            taskGenerations[taskID] == generation
        else {
            return false
        }

        if activityCreationInProgress[taskID]
            == generation {
            return false
        }

        activityCreationInProgress[taskID] =
            generation

        return true
    }

    private func endActivityCreation(
        taskID: String,
        generation: UUID
    ) {
        stateLock.lock()

        if activityCreationInProgress[taskID]
            == generation {
            activityCreationInProgress
                .removeValue(
                    forKey: taskID
                )
        }

        stateLock.unlock()
    }

    private func shouldHandleProgress(
        taskID: String,
        generation: UUID,
        now: Date
    ) -> Bool {
        stateLock.lock()
        defer { stateLock.unlock() }

        guard
            !suppressedTaskIDs.contains(taskID),
            taskGenerations[taskID]
                == generation
        else {
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

    // MARK: - Gopeed event entry point

    @objc
    func handleTaskEventPayload(_ payload: String) {

        guard #available(iOS 16.2, *) else {
            return
        }

        guard let event = GopeedTaskEvent.decode(payload) else {
            print("LiveActivity: invalid task event payload")
            return
        }

        let taskID = event.taskID

        guard !isTaskSuppressed(taskID) else {
            return
        }

        let name = event.name

        switch event.type {

        case .start:
            let generation =
                startGeneration(
                    taskID: taskID
                )

            Task {
                await refreshActivity(
                    taskID: taskID,
                    name: name,
                    generation: generation,
                    allowStart: true
                )
            }

        case .progress:
            guard
                let generation =
                    generationForProgress(
                        taskID: taskID
                    )
            else {
                return
            }

            handleProgress(
                taskID: taskID,
                name: name,
                generation: generation
            )

        case .pause:
            if let generation =
                invalidateGeneration(
                    taskID: taskID
                ) {
                Task {
                    await removeActivity(
                        taskID: taskID,
                        generation: generation,
                        reason: "paused"
                    )
                }
            }

        case .done:
            if let generation =
                invalidateGeneration(
                    taskID: taskID
                ) {
                Task {
                    await finishActivity(
                        taskID: taskID,
                        generation: generation
                    )
                }
            }

        case .error:
            let error =
                event.error
                ?? "Download failed"

            if let generation =
                invalidateGeneration(
                    taskID: taskID
                ) {
                Task {
                    await failActivity(
                        taskID: taskID,
                        generation: generation,
                        error: error
                    )
                }
            }

        case .delete:
            if let generation =
                invalidateGeneration(
                    taskID: taskID
                ) {
                Task {
                    await removeActivity(
                        taskID: taskID,
                        generation: generation,
                        reason: "removed"
                    )
                }
            }

        case .other:
            break
        }
    }


    // MARK: - Progress throttling

    @available(iOS 16.2, *)
    private func handleProgress(
        taskID: String,
        name: String,
        generation: UUID
    ) {
        let now = Date()

        guard shouldHandleProgress(
            taskID: taskID,
            generation: generation,
            now: now
        ) else {
            return
        }

        Task {
            await refreshActivity(
                taskID: taskID,
                name: name,
                generation: generation,
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
    private func findActivities(
        taskID: String,
        generation: UUID
    ) -> [Activity<GopeedDownloadAttributes>] {
        let generationID =
            generation.uuidString

        return findActivities(
            taskID: taskID
        )
        .filter {
            $0.attributes.generation
                == generationID
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

    @available(iOS 16.2, *)
    private func removeStaleActivities(
        taskID: String,
        keeping generation: UUID
    ) async {
        let generationID =
            generation.uuidString

        let stale =
            findActivities(taskID: taskID)
                .filter {
                    $0.attributes.generation
                        != generationID
                }

        for activity in stale {
            await activity.end(
                nil,
                dismissalPolicy: .immediate
            )
        }
    }


    // MARK: - Start / update Activity

    @available(iOS 16.2, *)
    private func refreshActivity(
        taskID: String,
        name: String,
        generation: UUID,
        allowStart: Bool
    ) async {
        guard
            !isTaskSuppressed(taskID),
            isCurrentGeneration(
                taskID: taskID,
                generation: generation
            )
        else {
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

        guard
            !isTaskSuppressed(taskID),
            isCurrentGeneration(
                taskID: taskID,
                generation: generation
            )
        else {
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
                taskID: taskID,
                generation: generation
            )

        if !existing.isEmpty {
            await collapseDuplicateActivities(
                existing,
                content: content
            )
            return
        }

        guard
            allowStart,
            runtime.status == "running"
        else {
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

        guard
            appIsActive,
            !isTaskSuppressed(taskID),
            isCurrentGeneration(
                taskID: taskID,
                generation: generation
            )
        else {
            return
        }

        guard beginActivityCreation(
            taskID: taskID,
            generation: generation
        ) else {
            return
        }

        defer {
            endActivityCreation(
                taskID: taskID,
                generation: generation
            )
        }

        guard
            !isTaskSuppressed(taskID),
            isCurrentGeneration(
                taskID: taskID,
                generation: generation
            )
        else {
            return
        }

        await removeStaleActivities(
            taskID: taskID,
            keeping: generation
        )

        guard
            !isTaskSuppressed(taskID),
            isCurrentGeneration(
                taskID: taskID,
                generation: generation
            )
        else {
            return
        }

        let recheck =
            findActivities(
                taskID: taskID,
                generation: generation
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
                fileName: name,
                generation:
                    generation.uuidString
            )

        do {
            let activity =
                try Activity.request(
                    attributes: attributes,
                    content: content,
                    pushType: nil
                )

            if
                isTaskSuppressed(taskID)
                || !isCurrentGeneration(
                    taskID: taskID,
                    generation: generation
                )
            {
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
        taskID: String,
        generation: UUID
    ) async {
        let activities =
            findActivities(
                taskID: taskID,
                generation: generation
            )

        guard !activities.isEmpty else {
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

        print(
            "LiveActivity: completed",
            taskID
        )
    }


    // MARK: - Error Activity

    @available(iOS 16.2, *)
    private func failActivity(
        taskID: String,
        generation: UUID,
        error: String
    ) async {
        let activities =
            findActivities(
                taskID: taskID,
                generation: generation
            )

        guard !activities.isEmpty else {
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

        print(
            "LiveActivity: task failed:",
            taskID,
            error
        )
    }


    // MARK: - Delete / suppression cleanup

    @available(iOS 16.2, *)
    private func removeActivity(
        taskID: String,
        generation: UUID,
        reason: String
    ) async {
        let activities =
            findActivities(
                taskID: taskID,
                generation: generation
            )

        for activity in activities {
            await activity.end(
                nil,
                dismissalPolicy: .immediate
            )
        }

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
