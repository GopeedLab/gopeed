import 'dart:math' as math;

import 'package:flutter/material.dart' show Icons, Scrollbar, ScrollbarOrientation;
import 'package:flutter/widgets.dart';

import '../../../../../api/model/task_stats.dart';
import '../../../../../core/utils/byte_size_formatter.dart';
import '../../../../../shared/theme/app_palette.dart';
import '../../../../../l10n/l10n.dart';

enum PeerTableProtocol { bt, ed2k }

enum _PeerSortColumn { address, client, transport, downloadSpeed, uploadSpeed, pieces, completion, relevance, source }

class PeerTable extends StatefulWidget {
  const PeerTable({super.key, required this.protocol, required this.peers});

  final PeerTableProtocol protocol;
  final List<TaskPeerStats> peers;

  @override
  State<PeerTable> createState() => _PeerTableState();
}

class _PeerTableState extends State<PeerTable> {
  static const _headerHeight = 40.0;
  static const _rowHeight = 38.0;

  double get _tableWidth => widget.protocol == PeerTableProtocol.bt ? 1070 : 870;

  final ScrollController _horizontalController = ScrollController();
  final ScrollController _verticalController = ScrollController();
  _PeerSortColumn _sortColumn = _PeerSortColumn.downloadSpeed;
  bool _ascending = false;

  @override
  void dispose() {
    _horizontalController.dispose();
    _verticalController.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final palette = AppPalette.of(context);
    final peers = [...widget.peers]..sort(_comparePeers);
    return SizedBox(
      height: 300,
      child: LayoutBuilder(
        builder: (context, constraints) {
          final contentWidth = math.max(_tableWidth, constraints.maxWidth);
          return Scrollbar(
            controller: _horizontalController,
            thumbVisibility: true,
            trackVisibility: true,
            interactive: true,
            thickness: 7,
            scrollbarOrientation: ScrollbarOrientation.bottom,
            child: SingleChildScrollView(
              controller: _horizontalController,
              scrollDirection: Axis.horizontal,
              child: SizedBox(
                width: contentWidth,
                height: constraints.maxHeight,
                child: Column(
                  children: [
                    Container(
                      height: _headerHeight,
                      decoration: BoxDecoration(
                        color: palette.sideBg,
                        border: Border(bottom: BorderSide(color: palette.border)),
                      ),
                      child: Row(children: _headers(context)),
                    ),
                    Expanded(
                      child: ListView.builder(
                        controller: _verticalController,
                        padding: const EdgeInsets.only(bottom: 12),
                        itemCount: peers.length,
                        itemExtent: _rowHeight,
                        itemBuilder: (context, index) => _PeerRow(protocol: widget.protocol, peer: peers[index]),
                      ),
                    ),
                  ],
                ),
              ),
            ),
          );
        },
      ),
    );
  }

  List<Widget> _headers(BuildContext context) => [
    _header(context.l10n.address, _PeerSortColumn.address, 180),
    _header(context.l10n.client, _PeerSortColumn.client, 150),
    _header(context.l10n.protocol, _PeerSortColumn.transport, 80),
    _header(context.l10n.speed, _PeerSortColumn.downloadSpeed, 120),
    _header(context.l10n.uploadSpeed, _PeerSortColumn.uploadSpeed, 120),
    _header(context.l10n.pieces, _PeerSortColumn.pieces, 100),
    if (widget.protocol == PeerTableProtocol.bt) ...[
      _header(context.l10n.progress, _PeerSortColumn.completion, 100),
      _header(context.l10n.fileRelevance, _PeerSortColumn.relevance, 100),
    ],
    _header(context.l10n.source, _PeerSortColumn.source, 120),
  ];

  Widget _header(String label, _PeerSortColumn column, double width) {
    final palette = AppPalette.of(context);
    final active = _sortColumn == column;
    return GestureDetector(
      behavior: HitTestBehavior.opaque,
      onTap: () {
        setState(() {
          if (_sortColumn == column) {
            _ascending = !_ascending;
          } else {
            _sortColumn = column;
            _ascending = column != _PeerSortColumn.downloadSpeed && column != _PeerSortColumn.uploadSpeed;
          }
        });
      },
      child: SizedBox(
        width: width,
        child: Padding(
          padding: const EdgeInsets.symmetric(horizontal: 10),
          child: Row(
            children: [
              Expanded(
                child: Text(
                  label,
                  maxLines: 1,
                  overflow: TextOverflow.ellipsis,
                  style: TextStyle(
                    color: active ? palette.textPrimary : palette.textSecondary,
                    fontSize: 11,
                    fontWeight: FontWeight.w700,
                  ),
                ),
              ),
              if (active) ...[
                const SizedBox(width: 3),
                Icon(_ascending ? Icons.arrow_upward : Icons.arrow_downward, size: 12, color: palette.textMuted),
              ],
            ],
          ),
        ),
      ),
    );
  }

  int _comparePeers(TaskPeerStats a, TaskPeerStats b) {
    final result = switch (_sortColumn) {
      _PeerSortColumn.address => a.address.compareTo(b.address),
      _PeerSortColumn.client => a.client.compareTo(b.client),
      _PeerSortColumn.transport => a.transport.compareTo(b.transport),
      _PeerSortColumn.downloadSpeed => a.downloadSpeed.compareTo(b.downloadSpeed),
      _PeerSortColumn.uploadSpeed => a.uploadSpeed.compareTo(b.uploadSpeed),
      _PeerSortColumn.pieces => a.pieceCount.compareTo(b.pieceCount),
      _PeerSortColumn.completion => (a.completion ?? -1).compareTo(b.completion ?? -1),
      _PeerSortColumn.relevance => (a.relevance ?? -1).compareTo(b.relevance ?? -1),
      _PeerSortColumn.source => a.source.compareTo(b.source),
    };
    return _ascending ? result : -result;
  }
}

class _PeerRow extends StatelessWidget {
  const _PeerRow({required this.protocol, required this.peer});

  final PeerTableProtocol protocol;
  final TaskPeerStats peer;

  @override
  Widget build(BuildContext context) {
    final palette = AppPalette.of(context);
    return DecoratedBox(
      decoration: BoxDecoration(
        border: Border(bottom: BorderSide(color: palette.border)),
      ),
      child: Row(
        children: [
          _cell(context, peer.address, 180),
          _cell(context, peer.client, 150),
          _cell(context, _formatProtocol(protocol, peer.transport), 80),
          _cell(context, _formatRate(peer.downloadSpeed), 120),
          _cell(context, _formatRate(peer.uploadSpeed), 120),
          _cell(context, peer.pieceCount.toString(), 100),
          if (protocol == PeerTableProtocol.bt) ...[
            _cell(context, _formatPercentage(peer.completion), 100),
            _cell(context, _formatPercentage(peer.relevance), 100),
          ],
          _cell(context, peer.source, 120),
        ],
      ),
    );
  }

  Widget _cell(BuildContext context, String value, double width) {
    final palette = AppPalette.of(context);
    return SizedBox(
      width: width,
      child: Padding(
        padding: const EdgeInsets.symmetric(horizontal: 10),
        child: Text(
          value,
          maxLines: 1,
          overflow: TextOverflow.ellipsis,
          style: TextStyle(color: palette.textSecondary, fontSize: 11),
        ),
      ),
    );
  }
}

String _formatRate(int bytesPerSecond) => bytesPerSecond <= 0 ? '—' : '${_formatBytes(bytesPerSecond)}/s';

String _formatProtocol(PeerTableProtocol protocol, String transport) {
  if (protocol == PeerTableProtocol.ed2k) return 'ED2K';
  return switch (transport.toLowerCase()) {
    'utp' => 'uTP',
    'webrtc' => 'WebRTC',
    _ => 'BT',
  };
}

String _formatPercentage(double? ratio) {
  if (ratio == null) return '—';
  final percentage = ratio.clamp(0.0, 1.0) * 100;
  final fractionDigits = percentage == percentage.roundToDouble() ? 0 : 1;
  return '${percentage.toStringAsFixed(fractionDigits)}%';
}

String _formatBytes(int bytes) {
  return ByteSizeFormatter.format(bytes);
}
