import XCTest

@testable import Runner

final class RunnerTests: XCTestCase {

    func testDecodesKnownTaskEventTypes() throws {
        let cases: [(String, GopeedTaskEventType)] = [
            ("task.start", .start),
            ("task.progress", .progress),
            ("task.pause", .pause),
            ("task.done", .done),
            ("task.error", .error),
            ("task.delete", .delete),
        ]

        for (rawType, expectedType) in cases {
            let payload =
                """
                {
                  "type": "\(rawType)",
                  "taskId": "task-1",
                  "name": "Test Download"
                }
                """

            let event = try XCTUnwrap(
                GopeedTaskEvent.decode(payload)
            )

            XCTAssertEqual(
                event.type,
                expectedType
            )
            XCTAssertEqual(
                event.taskID,
                "task-1"
            )
            XCTAssertEqual(
                event.name,
                "Test Download"
            )
        }
    }

    func testUnknownEventTypeIsPreserved() throws {
        let payload =
            """
            {
              "type": "task.future-event",
              "taskId": "task-2"
            }
            """

        let event = try XCTUnwrap(
            GopeedTaskEvent.decode(payload)
        )

        XCTAssertEqual(
            event.type,
            .other("task.future-event")
        )
    }

    func testMissingNameUsesDefault() throws {
        let payload =
            """
            {
              "type": "task.start",
              "taskId": "task-3"
            }
            """

        let event = try XCTUnwrap(
            GopeedTaskEvent.decode(payload)
        )

        XCTAssertEqual(
            event.name,
            "Download"
        )
    }

    func testErrorMessageIsDecoded() throws {
        let payload =
            """
            {
              "type": "task.error",
              "taskId": "task-4",
              "name": "Failed Download",
              "error": "Network unavailable"
            }
            """

        let event = try XCTUnwrap(
            GopeedTaskEvent.decode(payload)
        )

        XCTAssertEqual(
            event.type,
            .error
        )
        XCTAssertEqual(
            event.error,
            "Network unavailable"
        )
    }

    func testMissingErrorRemainsNil() throws {
        let payload =
            """
            {
              "type": "task.error",
              "taskId": "task-5"
            }
            """

        let event = try XCTUnwrap(
            GopeedTaskEvent.decode(payload)
        )

        XCTAssertNil(event.error)
    }

    func testMalformedJSONReturnsNil() {
        let payload =
            """
            {
              "type": "task.start",
            """

        XCTAssertNil(
            GopeedTaskEvent.decode(payload)
        )
    }

    func testMissingTypeReturnsNil() {
        let payload =
            """
            {
              "taskId": "task-6"
            }
            """

        XCTAssertNil(
            GopeedTaskEvent.decode(payload)
        )
    }

    func testMissingTaskIDReturnsNil() {
        let payload =
            """
            {
              "type": "task.start"
            }
            """

        XCTAssertNil(
            GopeedTaskEvent.decode(payload)
        )
    }
}