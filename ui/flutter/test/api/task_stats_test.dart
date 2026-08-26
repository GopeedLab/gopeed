import 'package:flutter_test/flutter_test.dart';
import 'package:gopeed/api/model/task.dart';
import 'package:gopeed/api/model/task_stats.dart';

void main() {
  test('HTTP stats parse the current Go connection payload', () {
    final stats =
        TaskStats.fromJson(Protocol.http, {
              'snapshot': {
                'connections': [
                  {'downloaded': 1024, 'total': 4096, 'completed': false, 'failed': false, 'retryTimes': 2},
                  {'downloaded': 2048, 'total': 2048, 'completed': true, 'failed': false, 'retryTimes': 0},
                ],
              },
              'runtime': null,
            })
            as HttpTaskStats;

    expect(stats.connections, hasLength(2));
    expect(stats.connections.first.downloaded, 1024);
    expect(stats.connections.first.total, 4096);
    expect(stats.connections.first.retryTimes, 2);
    expect(stats.connections.last.completed, isTrue);
  });

  test('BT stats decode the ordered completion bitset and peer transport', () {
    final stats =
        TaskStats.fromJson(Protocol.bt, {
              'snapshot': {
                'seedBytes': 820,
                'seedRatio': 0.64,
                'pieceMap': {
                  'encoding': 'bitset-v1',
                  'pieceCount': 8,
                  'pieceSize': 16384,
                  'completedPieces': 3,
                  'data': 'hQ==',
                },
              },
              'runtime': {
                'totalPeers': 128,
                'activePeers': 16,
                'connectedSeeders': 8,
                'connectedLeechers': 8,
                'peers': [
                  {
                    'address': '127.0.0.1:6881',
                    'client': 'Gopeed',
                    'downloadSpeed': 2048,
                    'uploadSpeed': 1024,
                    'pieceCount': 12,
                    'completion': 0.75,
                    'relevance': 0.25,
                    'source': 'PEX',
                    'transport': 'utp',
                  },
                ],
              },
            })
            as BtTaskStats;

    expect(stats.activePeers, 16);
    expect(stats.connectedLeechers, 8);
    expect(stats.pieceMap.toStates(), [
      TaskPieceState.completed,
      TaskPieceState.pending,
      TaskPieceState.completed,
      TaskPieceState.pending,
      TaskPieceState.pending,
      TaskPieceState.pending,
      TaskPieceState.pending,
      TaskPieceState.completed,
    ]);
    expect(stats.pieceMap.completedPieces, 3);
    expect(stats.pieceMap.pieceSize, 16384);
    expect(stats.peers.single.downloadSpeed, 2048);
    expect(stats.peers.single.pieceCount, 12);
    expect(stats.peers.single.completion, 0.75);
    expect(stats.peers.single.relevance, 0.25);
    expect(stats.peers.single.transport, 'utp');
  });

  test('BT persisted snapshot remains useful when runtime is absent', () {
    final stats =
        TaskStats.fromJson(Protocol.bt, {
              'snapshot': {
                'seedBytes': 820,
                'seedRatio': 0.64,
                'pieceMap': {
                  'encoding': 'bitset-v1',
                  'pieceCount': 4,
                  'pieceSize': 16384,
                  'completedPieces': 2,
                  'data': 'Aw==',
                },
              },
              'runtime': null,
            })
            as BtTaskStats;

    expect(stats.totalPeers, 0);
    expect(stats.activePeers, 0);
    expect(stats.connectedLeechers, 0);
    expect(stats.peers, isEmpty);
    expect(stats.pieceMap.completedPieces, 2);
    expect(stats.pieceMap.toStates(), [
      TaskPieceState.completed,
      TaskPieceState.completed,
      TaskPieceState.pending,
      TaskPieceState.pending,
    ]);
  });

  test('piece map rejects unsupported versions and malformed byte lengths', () {
    expect(
      TaskPieceMap.fromJson({
        'encoding': 'bitset-v2',
        'pieceCount': 4,
        'pieceSize': 1024,
        'completedPieces': 0,
        'data': 'AA==',
      }),
      isNull,
    );
    expect(
      TaskPieceMap.fromJson({
        'encoding': 'bitset-v1',
        'pieceCount': 9,
        'pieceSize': 1024,
        'completedPieces': 0,
        'data': 'AA==',
      }),
      isNull,
    );
  });

  test('stats models ignore fields that are not part of the Go response contract', () {
    final http =
        TaskStats.fromJson(Protocol.http, {
              'snapshot': {
                'connections': [
                  {'received': 1024, 'completed': false, 'failed': false, 'retries': 2},
                ],
              },
            })
            as HttpTaskStats;
    final bt =
        TaskStats.fromJson(Protocol.bt, {
              'snapshot': {
                'uploaded': 1024,
                'shareRatio': 0.5,
                'pieces': [2, 1, 0],
              },
              'runtime': {'seeders': 8},
            })
            as BtTaskStats;

    expect(http.connections.single.downloaded, 0);
    expect(http.connections.single.retryTimes, 0);
    expect(bt.connectedSeeders, 0);
    expect(bt.seedBytes, 0);
    expect(bt.seedRatio, 0);
    expect(bt.pieceMap.isEmpty, isTrue);
  });

  test('ED2K stats parse the existing aggregate fields', () {
    final stats =
        TaskStats.fromJson(Protocol.ed2k, {
              'snapshot': {'upload': 320, 'totalDone': 68, 'totalWanted': 100},
              'runtime': {
                'serverIdClass': 'high',
                'activePeers': 12,
                'totalPeers': 86,
                'downloadRate': 4800000,
                'uploadRate': 320000,
              },
            })
            as Ed2kTaskStats;

    expect(stats.totalPeers, 86);
    expect(stats.serverIdClass, 'high');
    expect(stats.downloadRate, 4800000);
    expect(stats.totalDone / stats.totalWanted, 0.68);
    expect(stats.peers, isEmpty);
  });

  test('ED2K server ID class defaults to unknown', () {
    final stats =
        TaskStats.fromJson(Protocol.ed2k, {
              'runtime': {'activePeers': 0},
            })
            as Ed2kTaskStats;

    expect(stats.serverIdClass, 'unknown');
  });

  test('unsupported protocols have no statistics panel', () {
    expect(
      TaskStats.fromJson(Protocol.gblob, {
        'snapshot': {'value': 1},
      }),
      isNull,
    );
    expect(
      TaskStats.fromJson(null, {
        'snapshot': {'value': 1},
      }),
      isNull,
    );
  });
}
