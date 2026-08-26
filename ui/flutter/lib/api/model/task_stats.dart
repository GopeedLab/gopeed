export 'piece_map_codec.dart' show TaskPieceMap, TaskPieceState;

import 'piece_map_codec.dart';
import 'task.dart';

class TaskPeerStats {
  const TaskPeerStats({
    required this.address,
    required this.client,
    required this.downloadSpeed,
    required this.uploadSpeed,
    required this.pieceCount,
    required this.completion,
    required this.relevance,
    required this.source,
    required this.transport,
  });

  final String address;
  final String client;
  final int downloadSpeed;
  final int uploadSpeed;
  final int pieceCount;
  final double? completion;
  final double? relevance;
  final String source;
  final String transport;

  factory TaskPeerStats.fromJson(Map<String, dynamic> json) {
    return TaskPeerStats(
      address: _string(json, 'address'),
      client: _string(json, 'client'),
      downloadSpeed: _integer(json, 'downloadSpeed'),
      uploadSpeed: _integer(json, 'uploadSpeed'),
      pieceCount: _integer(json, 'pieceCount'),
      completion: _optionalDecimal(json, 'completion'),
      relevance: _optionalDecimal(json, 'relevance'),
      source: _string(json, 'source'),
      transport: _string(json, 'transport'),
    );
  }
}

sealed class TaskStats {
  const TaskStats({required this.peers, required this.pieceMap});

  final List<TaskPeerStats> peers;
  final TaskPieceMap pieceMap;

  static TaskStats? fromJson(Protocol? protocol, Map<String, dynamic> json) {
    final snapshot = _map(json['snapshot']);
    final runtime = _map(json['runtime']);
    if (snapshot.isEmpty && runtime.isEmpty) return null;
    return switch (protocol) {
      Protocol.http => HttpTaskStats.fromJson(snapshot),
      Protocol.bt => BtTaskStats.fromJson(snapshot: snapshot, runtime: runtime),
      Protocol.ed2k => Ed2kTaskStats.fromJson(snapshot: snapshot, runtime: runtime),
      Protocol.gblob || null => null,
    };
  }
}

class HttpConnectionStats {
  const HttpConnectionStats({
    required this.downloaded,
    required this.total,
    required this.completed,
    required this.failed,
    required this.retryTimes,
  });

  final int downloaded;
  final int total;
  final bool completed;
  final bool failed;
  final int retryTimes;

  factory HttpConnectionStats.fromJson(Map<String, dynamic> json) {
    return HttpConnectionStats(
      downloaded: _integer(json, 'downloaded'),
      total: _integer(json, 'total'),
      completed: json['completed'] == true,
      failed: json['failed'] == true,
      retryTimes: _integer(json, 'retryTimes'),
    );
  }
}

class HttpTaskStats extends TaskStats {
  HttpTaskStats({required this.connections}) : super(peers: const [], pieceMap: TaskPieceMap.empty());

  final List<HttpConnectionStats> connections;

  factory HttpTaskStats.fromJson(Map<String, dynamic> json) {
    return HttpTaskStats(
      connections: _maps(json['connections']).map(HttpConnectionStats.fromJson).toList(growable: false),
    );
  }
}

class BtTaskStats extends TaskStats {
  const BtTaskStats({
    required this.totalPeers,
    required this.activePeers,
    required this.connectedSeeders,
    required this.connectedLeechers,
    required this.seedBytes,
    required this.seedRatio,
    required super.peers,
    required super.pieceMap,
  });

  final int totalPeers;
  final int activePeers;
  final int connectedSeeders;
  final int connectedLeechers;
  final int seedBytes;
  final double seedRatio;

  factory BtTaskStats.fromJson({required Map<String, dynamic> snapshot, required Map<String, dynamic> runtime}) {
    return BtTaskStats(
      totalPeers: _integer(runtime, 'totalPeers'),
      activePeers: _integer(runtime, 'activePeers'),
      connectedSeeders: _integer(runtime, 'connectedSeeders'),
      connectedLeechers: _integer(runtime, 'connectedLeechers'),
      seedBytes: _integer(snapshot, 'seedBytes'),
      seedRatio: _decimal(snapshot, 'seedRatio'),
      peers: _peerList(runtime['peers']),
      pieceMap: TaskPieceMap.fromJson(snapshot['pieceMap']) ?? TaskPieceMap.empty(),
    );
  }
}

class Ed2kTaskStats extends TaskStats {
  const Ed2kTaskStats({
    required this.serverIdClass,
    required this.activePeers,
    required this.totalPeers,
    required this.downloadRate,
    required this.upload,
    required this.uploadRate,
    required this.totalDone,
    required this.totalWanted,
    required super.peers,
    required super.pieceMap,
  });

  final String serverIdClass;
  final int activePeers;
  final int totalPeers;
  final int downloadRate;
  final int upload;
  final int uploadRate;
  final int totalDone;
  final int totalWanted;
  int get completedPieces => pieceMap.completedPieces;

  factory Ed2kTaskStats.fromJson({required Map<String, dynamic> snapshot, required Map<String, dynamic> runtime}) {
    return Ed2kTaskStats(
      serverIdClass: _stringValue(runtime, 'serverIdClass', fallback: 'unknown'),
      activePeers: _integer(runtime, 'activePeers'),
      totalPeers: _integer(runtime, 'totalPeers'),
      downloadRate: _integer(runtime, 'downloadRate'),
      upload: _integer(snapshot, 'upload'),
      uploadRate: _integer(runtime, 'uploadRate'),
      totalDone: _integer(snapshot, 'totalDone'),
      totalWanted: _integer(snapshot, 'totalWanted'),
      peers: _peerList(runtime['peers']),
      pieceMap: TaskPieceMap.fromJson(snapshot['pieceMap']) ?? TaskPieceMap.empty(),
    );
  }
}

List<Map<String, dynamic>> _maps(Object? value) {
  if (value is! List) return const [];
  return value
      .whereType<Map>()
      .map((item) => {for (final entry in item.entries) entry.key.toString(): entry.value})
      .toList(growable: false);
}

Map<String, dynamic> _map(Object? value) {
  if (value is! Map) return const {};
  return {for (final entry in value.entries) entry.key.toString(): entry.value};
}

List<TaskPeerStats> _peerList(Object? value) => _maps(value).map(TaskPeerStats.fromJson).toList(growable: false);

int _integer(Map<String, dynamic> json, String key) => switch (json[key]) {
  final int value => value,
  _ => 0,
};

double _decimal(Map<String, dynamic> json, String key) => switch (json[key]) {
  final num value => value.toDouble(),
  _ => 0,
};

double? _optionalDecimal(Map<String, dynamic> json, String key) => switch (json[key]) {
  final num value => value.toDouble(),
  _ => null,
};

String _string(Map<String, dynamic> json, String key) => switch (json[key]) {
  final String value when value.isNotEmpty => value,
  _ => '—',
};

String _stringValue(Map<String, dynamic> json, String key, {required String fallback}) => switch (json[key]) {
  final String value when value.isNotEmpty => value,
  _ => fallback,
};
