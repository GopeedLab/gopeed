import 'package:json_annotation/json_annotation.dart';

part 'server_config.g.dart';

@JsonSerializable()
class ServerConfig {
  late String host;
  late int port;
  late int connections;
  late String downloadDir;
  Map<String, dynamic>? extra;

  ServerConfig({
    required this.host,
    required this.port,
    required this.connections,
    required this.downloadDir,
    this.extra,
  });

  factory ServerConfig.fromJson(Map<String, dynamic> json) =>
      _$ServerConfigFromJson(json);
  Map<String, dynamic> toJson() => _$ServerConfigToJson(this);
}
