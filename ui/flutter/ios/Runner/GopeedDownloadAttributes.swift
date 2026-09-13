import ActivityKit
import Foundation

@available(iOS 16.2, *)
struct GopeedDownloadAttributes: ActivityAttributes {

    struct ContentState: Codable, Hashable {

        // Real confirmed progress.
        var progress: Double

        var downloaded: Int64
        var total: Int64
        var speed: Int64

        var status: String

        // Used by the self-moving estimated progress bar.
        var estimatedStart: Date
        var estimatedEnd: Date
        var usesEstimatedProgress: Bool
    }

    // Static properties.
    var taskId: String
    var fileName: String
}
