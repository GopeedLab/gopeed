import 'package:json_annotation/json_annotation.dart';

part 'stats.g.dart';

@JsonSerializable(explicitToJson: true)
class Stats {
  List<StatsConnection> connections;
  String sha256;
  String crc32;
  int fileSize;
  int expectedSize;
  bool integrityVerified;

  Stats({
    required this.connections,
    required this.sha256,
    required this.crc32,
    required this.fileSize,
    required this.expectedSize,
    required this.integrityVerified,
  });

  factory Stats.fromJson(Map<String, dynamic> json) => _$StatsFromJson(json);

  Map<String, dynamic> toJson() => _$StatsToJson(this);
}

@JsonSerializable()
class StatsConnection {
  int downloaded;
  bool completed;
  bool failed;
  int retryTimes;

  StatsConnection({
    required this.downloaded,
    required this.completed,
    required this.failed,
    required this.retryTimes,
  });

  factory StatsConnection.fromJson(Map<String, dynamic> json) =>
      _$StatsConnectionFromJson(json);

  Map<String, dynamic> toJson() => _$StatsConnectionToJson(this);
}
