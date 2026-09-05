import 'dart:math' as math;

import 'package:flutter/gestures.dart';
import 'package:flutter/widgets.dart';

import '../../../../../api/model/task_stats.dart';
import '../../../../../core/utils/byte_size_formatter.dart';
import '../../../../../shared/theme/app_palette.dart';
import '../../../../../l10n/l10n.dart';

class PieceMap extends StatefulWidget {
  const PieceMap({super.key, required this.pieceMap, this.totalBytes});

  final TaskPieceMap pieceMap;
  final int? totalBytes;

  @override
  State<PieceMap> createState() => _PieceMapState();
}

class _PieceMapState extends State<PieceMap> {
  static const _gap = 3.0;
  static const _targetCellSize = 11.0;
  // The default desktop drawer fits 28 columns, so 252 cells fill 9 rows.
  static const _maxPaintedCells = 252;

  int? _focusedCell;
  PointerDeviceKind? _lastPointerKind;

  @override
  void didUpdateWidget(covariant PieceMap oldWidget) {
    super.didUpdateWidget(oldWidget);
    if (_focusedCell != null && _focusedCell! >= math.min(widget.pieceMap.pieceCount, _maxPaintedCells)) {
      _focusedCell = null;
    }
  }

  List<TaskPieceState> _displayPieces(int displayCount) {
    final pieceMap = widget.pieceMap;
    if (pieceMap.pieceCount <= displayCount) {
      return List.generate(pieceMap.pieceCount, pieceMap.stateAt, growable: false);
    }
    return List.generate(displayCount, (cell) {
      final start = cell * pieceMap.pieceCount ~/ displayCount;
      final end = math.max(start + 1, (cell + 1) * pieceMap.pieceCount ~/ displayCount);
      var allCompleted = true;
      for (var index = start; index < math.min(end, pieceMap.pieceCount); index++) {
        final state = pieceMap.stateAt(index);
        if (state != TaskPieceState.completed) allCompleted = false;
      }
      if (allCompleted) return TaskPieceState.completed;
      return TaskPieceState.pending;
    });
  }

  @override
  Widget build(BuildContext context) {
    final palette = AppPalette.of(context);
    return LayoutBuilder(
      builder: (context, constraints) {
        if (widget.pieceMap.isEmpty) return const SizedBox.shrink();
        final width = constraints.maxWidth;
        final columns = math.max(1, ((width + _gap) / (_targetCellSize + _gap)).floor());
        final displayCount = math.min(widget.pieceMap.pieceCount, _maxPaintedCells);
        final pieces = _displayPieces(displayCount);
        final focusedCell = _focusedCell != null && _focusedCell! < pieces.length ? _focusedCell : null;
        final cellSize = (width - _gap * (columns - 1)) / columns;
        final rows = (pieces.length / columns).ceil();
        final mapHeight = rows * cellSize + math.max(0, rows - 1) * _gap;

        int? cellAt(Offset offset) {
          final column = (offset.dx / (cellSize + _gap)).floor();
          final row = (offset.dy / (cellSize + _gap)).floor();
          if (column < 0 || column >= columns || row < 0 || row >= rows) return null;
          final localX = offset.dx - column * (cellSize + _gap);
          final localY = offset.dy - row * (cellSize + _gap);
          if (localX > cellSize || localY > cellSize) return null;
          final index = row * columns + column;
          return index < pieces.length ? index : null;
        }

        void updateFocus(PointerEvent event, {required bool pressed}) {
          final index = cellAt(event.localPosition);
          if (pressed || event.kind == PointerDeviceKind.mouse || event.kind == PointerDeviceKind.trackpad) {
            setState(() {
              _focusedCell = index;
              _lastPointerKind = event.kind;
            });
          }
        }

        return Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            MouseRegion(
              onExit: (_) {
                if (_lastPointerKind == PointerDeviceKind.mouse || _lastPointerKind == PointerDeviceKind.trackpad) {
                  setState(() => _focusedCell = null);
                }
              },
              child: Listener(
                behavior: HitTestBehavior.opaque,
                onPointerHover: (event) => updateFocus(event, pressed: false),
                onPointerDown: (event) => updateFocus(event, pressed: true),
                child: CustomPaint(
                  size: Size(width, mapHeight),
                  painter: _PieceMapPainter(
                    pieces: pieces,
                    columns: columns,
                    cellSize: cellSize,
                    gap: _gap,
                    palette: palette,
                    focusedCell: focusedCell,
                  ),
                ),
              ),
            ),
            SizedBox(
              height: 24,
              child: Padding(
                padding: const EdgeInsets.only(top: 6),
                child: Text(
                  focusedCell == null ? '' : _rangeLabel(focusedCell, pieces.length),
                  style: TextStyle(color: palette.textMuted, fontSize: 11),
                ),
              ),
            ),
          ],
        );
      },
    );
  }

  String _rangeLabel(int displayIndex, int displayCount) {
    final sourceCount = widget.pieceMap.pieceCount;
    final startPiece = displayIndex * sourceCount ~/ displayCount;
    final endPiece = math.max(startPiece + 1, (displayIndex + 1) * sourceCount ~/ displayCount) - 1;
    final pieceLabel = startPiece == endPiece
        ? context.l10n.piece(startPiece + 1)
        : context.l10n.pieceRange(startPiece + 1, endPiece + 1);
    final bytesPerPiece = widget.pieceMap.pieceSize;
    if (bytesPerPiece <= 0) return pieceLabel;
    final startByte = startPiece * bytesPerPiece;
    final rawEnd = (endPiece + 1) * bytesPerPiece - 1;
    final endByte = widget.totalBytes == null ? rawEnd : math.min(rawEnd, widget.totalBytes! - 1);
    return '$pieceLabel · ${_formatBytes(startByte)}–${_formatBytes(math.max(startByte, endByte))}';
  }
}

class _PieceMapPainter extends CustomPainter {
  const _PieceMapPainter({
    required this.pieces,
    required this.columns,
    required this.cellSize,
    required this.gap,
    required this.palette,
    required this.focusedCell,
  });

  final List<TaskPieceState> pieces;
  final int columns;
  final double cellSize;
  final double gap;
  final AppPalette palette;
  final int? focusedCell;

  @override
  void paint(Canvas canvas, Size size) {
    final paint = Paint();
    for (var index = 0; index < pieces.length; index++) {
      final column = index % columns;
      final row = index ~/ columns;
      final cellRect = Rect.fromLTWH(column * (cellSize + gap), row * (cellSize + gap), cellSize, cellSize);
      final rect = RRect.fromRectAndRadius(cellRect, const Radius.circular(2));
      final state = pieces[index];
      paint.color = switch (state) {
        TaskPieceState.pending => palette.surfaceSoft,
        TaskPieceState.completed => palette.success,
      };
      canvas.drawRRect(rect, paint);
      if (focusedCell == index) {
        paint
          ..style = PaintingStyle.stroke
          ..strokeWidth = 1.5
          ..color = palette.textPrimary;
        canvas.drawRRect(rect, paint);
        paint.style = PaintingStyle.fill;
      }
    }
  }

  @override
  bool shouldRepaint(covariant _PieceMapPainter oldDelegate) {
    return oldDelegate.pieces != pieces ||
        oldDelegate.palette != palette ||
        oldDelegate.focusedCell != focusedCell ||
        oldDelegate.columns != columns ||
        oldDelegate.cellSize != cellSize;
  }
}

String _formatBytes(int bytes) {
  return ByteSizeFormatter.format(bytes);
}
