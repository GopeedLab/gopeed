import Foundation

enum GopeedTaskEventType: Equatable {
    case start
    case progress
    case pause
    case done
    case error
    case delete
    case other(String)

    init(rawValue: String) {
        switch rawValue {
        case "task.start":
            self = .start
        case "task.progress":
            self = .progress
        case "task.pause":
            self = .pause
        case "task.done":
            self = .done
        case "task.error":
            self = .error
        case "task.delete":
            self = .delete
        default:
            self = .other(rawValue)
        }
    }
}

struct GopeedTaskEvent {
    let type: GopeedTaskEventType
    let taskID: String
    let name: String
    let error: String?

    private struct Payload: Decodable {
        let type: String
        let taskId: String
        let name: String?
        let error: String?
    }

    static func decode(
        _ payload: String
    ) -> GopeedTaskEvent? {
        guard
            let data = payload.data(using: .utf8),
            let decoded =
                try? JSONDecoder().decode(
                    Payload.self,
                    from: data
                )
        else {
            return nil
        }

        return GopeedTaskEvent(
            type:
                GopeedTaskEventType(
                    rawValue: decoded.type
                ),
            taskID: decoded.taskId,
            name: decoded.name ?? "Download",
            error: decoded.error
        )
    }
}
