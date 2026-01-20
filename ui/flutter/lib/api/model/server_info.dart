import 'package:json_annotation/json_annotation.dart';

part 'server_info.g.dart';

@JsonSerializable()
class ServerInfo {
  String version;
  String runtime;
  String os;
  String arch;
  bool inDocker;

  ServerInfo({
    required this.version,
    required this.runtime,
    required this.os,
    required this.arch,
    required this.inDocker,
  });

  factory ServerInfo.fromJson(Map<String, dynamic> json) =>
      _$ServerInfoFromJson(json);

  Map<String, dynamic> toJson() => _$ServerInfoToJson(this);
}

