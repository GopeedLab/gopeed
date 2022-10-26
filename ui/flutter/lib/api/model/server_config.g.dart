// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'server_config.dart';

// **************************************************************************
// JsonSerializableGenerator
// **************************************************************************

ServerConfig _$ServerConfigFromJson(Map<String, dynamic> json) => ServerConfig(
      host: json['host'] as String,
      port: json['port'] as int,
      connections: json['connections'] as int,
      downloadDir: json['downloadDir'] as String,
      extra: json['extra'] as Map<String, dynamic>?,
    );

Map<String, dynamic> _$ServerConfigToJson(ServerConfig instance) {
  final val = <String, dynamic>{
    'host': instance.host,
    'port': instance.port,
    'connections': instance.connections,
    'downloadDir': instance.downloadDir,
  };

  void writeNotNull(String key, dynamic value) {
    if (value != null) {
      val[key] = value;
    }
  }

  writeNotNull('extra', instance.extra);
  return val;
}
