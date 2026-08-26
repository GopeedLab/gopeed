import 'dart:convert';
import 'dart:typed_data';

enum TaskPieceState { pending, completed }

/// Decoded view of Gopeed's ordered `bitset-v1` piece map.
///
/// A set bit means the piece is complete and verified. The decoder keeps the
/// compact bytes instead of expanding large torrents into per-piece objects;
/// [stateAt] reads any piece in O(1).
class TaskPieceMap {
  const TaskPieceMap._({
    required this.encoding,
    required this.pieceCount,
    required this.pieceSize,
    required this.completedPieces,
    required Uint8List data,
  }) : _data = data;

  static const supportedEncoding = 'bitset-v1';

  final String encoding;
  final int pieceCount;
  final int pieceSize;
  final int completedPieces;
  final Uint8List _data;

  static TaskPieceMap empty() =>
      TaskPieceMap._(encoding: supportedEncoding, pieceCount: 0, pieceSize: 0, completedPieces: 0, data: Uint8List(0));

  /// Decodes the Base64 bitset returned by Go. Piece N uses bit `N % 8` of
  /// byte `N ~/ 8`, from least-significant to most-significant bit.
  static TaskPieceMap? fromJson(Object? value) {
    if (value is! Map) return null;
    final json = {for (final entry in value.entries) entry.key.toString(): entry.value};
    final encoding = json['encoding'];
    final pieceCount = json['pieceCount'];
    final pieceSize = json['pieceSize'];
    final completedPieces = json['completedPieces'];
    final encodedData = json['data'];
    if (encoding != supportedEncoding ||
        pieceCount is! int ||
        pieceCount < 0 ||
        pieceSize is! int ||
        pieceSize < 0 ||
        completedPieces is! int ||
        completedPieces < 0 ||
        completedPieces > pieceCount ||
        encodedData is! String) {
      return null;
    }

    try {
      final data = base64Decode(encodedData);
      final expectedLength = (pieceCount + 7) ~/ 8;
      if (data.length != expectedLength) return null;

      var actualCompleted = 0;
      for (var index = 0; index < pieceCount; index++) {
        if ((data[index ~/ 8] & (1 << (index % 8))) != 0) actualCompleted++;
      }
      if (actualCompleted != completedPieces) return null;

      return TaskPieceMap._(
        encoding: encoding,
        pieceCount: pieceCount,
        pieceSize: pieceSize,
        completedPieces: completedPieces,
        data: data,
      );
    } on FormatException {
      return null;
    }
  }

  /// Packs local states with the same bit order. Used for widget construction
  /// and tests; API responses are decoded through [fromJson].
  factory TaskPieceMap.fromStates(List<TaskPieceState> states, {int pieceSize = 0}) {
    final data = Uint8List((states.length + 7) ~/ 8);
    var completedPieces = 0;
    for (var index = 0; index < states.length; index++) {
      if (states[index] != TaskPieceState.completed) continue;
      data[index ~/ 8] |= 1 << (index % 8);
      completedPieces++;
    }
    return TaskPieceMap._(
      encoding: supportedEncoding,
      pieceCount: states.length,
      pieceSize: pieceSize,
      completedPieces: completedPieces,
      data: data,
    );
  }

  bool get isEmpty => pieceCount == 0;
  bool get isNotEmpty => !isEmpty;

  TaskPieceState stateAt(int index) {
    RangeError.checkValidIndex(index, this, 'index', pieceCount);
    final completed = (_data[index ~/ 8] & (1 << (index % 8))) != 0;
    return completed ? TaskPieceState.completed : TaskPieceState.pending;
  }

  bool contains(TaskPieceState state) => switch (state) {
    TaskPieceState.completed => completedPieces > 0,
    TaskPieceState.pending => completedPieces < pieceCount,
  };

  List<TaskPieceState> toStates() => List.generate(pieceCount, stateAt, growable: false);
}
