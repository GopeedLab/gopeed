import ActivityKit
import SwiftUI
import WidgetKit

@main
struct GopeedLiveActivityBundle: WidgetBundle {
    var body: some Widget {
        GopeedLiveActivityWidget()
    }
}

struct GopeedLiveActivityWidget: Widget {

    var body: some WidgetConfiguration {

        ActivityConfiguration(
            for: GopeedDownloadAttributes.self
        ) { context in

            // MARK: Lock Screen / notification banner

            VStack(spacing: 10) {

                HStack {

                    Image(systemName: statusIcon(context.state.status))
                        .font(.title3)

                    VStack(alignment: .leading, spacing: 2) {

                        Text(context.attributes.fileName)
                            .font(.headline)
                            .lineLimit(1)

                        Text(statusText(context.state.status))
                            .font(.caption)
                            .foregroundStyle(.secondary)
                    }

                    Spacer()

                    Text(
                        "\(Int(context.state.progress * 100))%"
                    )
                    .font(.headline)
                    .monospacedDigit()
                }

                progressView(context)

                HStack {

                    Text(
                        "\(formatBytes(context.state.downloaded)) / \(formatBytes(context.state.total))"
                    )

                    Spacer()

                    if context.state.speed > 0 {
                        Text(
                            "\(formatBytes(context.state.speed))/s"
                        )
                    }
                }
                .font(.caption)
                .foregroundStyle(.secondary)
            }
            .padding()
            .activityBackgroundTint(.black.opacity(0.85))
            .activitySystemActionForegroundColor(.white)

        } dynamicIsland: { context in

            DynamicIsland {

                // MARK: Expanded Dynamic Island

                DynamicIslandExpandedRegion(.leading) {
                    Image(
                        systemName: statusIcon(
                            context.state.status
                        )
                    )
                    .font(.title3)
                }

                DynamicIslandExpandedRegion(.center) {
                    Text(context.attributes.fileName)
                        .lineLimit(1)
                        .font(.headline)
                }

                DynamicIslandExpandedRegion(.trailing) {
                    Text(
                        "\(Int(context.state.progress * 100))%"
                    )
                    .monospacedDigit()
                    .font(.headline)
                }

                DynamicIslandExpandedRegion(.bottom) {

                    VStack(spacing: 6) {

                        progressView(context)

                        HStack {

                            Text(
                                "\(formatBytes(context.state.downloaded)) / \(formatBytes(context.state.total))"
                            )

                            Spacer()

                            if context.state.speed > 0 {
                                Text(
                                    "\(formatBytes(context.state.speed))/s"
                                )
                            }
                        }
                        .font(.caption2)
                        .foregroundStyle(.secondary)
                    }
                }

            } compactLeading: {

                Image(
                    systemName: statusIcon(
                        context.state.status
                    )
                )

            } compactTrailing: {

                Text(
                    "\(Int(context.state.progress * 100))%"
                )
                .monospacedDigit()

            } minimal: {

                Image(systemName: "arrow.down")
            }
        }
    }


    // MARK: - Progress bar

    @ViewBuilder
    private func progressView(
        _ context:
            ActivityViewContext<GopeedDownloadAttributes>
    ) -> some View {

        if context.state.usesEstimatedProgress,
           context.state.estimatedEnd >
            context.state.estimatedStart {

            ProgressView(
                timerInterval:
                    context.state.estimatedStart
                    ...
                    context.state.estimatedEnd,
                countsDown: false
            )

        } else {

            ProgressView(
                value: context.state.progress,
                total: 1.0
            )
        }
    }


    // MARK: - Formatting

    private func formatBytes(
        _ bytes: Int64
    ) -> String {

        guard bytes > 0 else {
            return "0 B"
        }

        return ByteCountFormatter.string(
            fromByteCount: bytes,
            countStyle: .file
        )
    }


    private func statusIcon(
        _ status: String
    ) -> String {

        switch status {

        case "done":
            return "checkmark.circle.fill"

        case "error":
            return "exclamationmark.triangle.fill"

        case "pause", "paused":
            return "pause.circle.fill"

        default:
            return "arrow.down.circle.fill"
        }
    }


    private func statusText(
        _ status: String
    ) -> String {

        switch status {

        case "done":
            return "Download complete"

        case "error":
            return "Download failed"

        case "pause", "paused":
            return "Paused"

        default:
            return "Downloading"
        }
    }
}
