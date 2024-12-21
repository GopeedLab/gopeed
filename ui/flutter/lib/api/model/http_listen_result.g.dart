// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'http_listen_result.dart';

// **************************************************************************
// JsonSerializableGenerator
// **************************************************************************

HttpListenResult _$HttpListenResultFromJson(Map<String, dynamic> json) =>
    HttpListenResult(
      host: json['host'] as String? ?? '',
      port: (json['port'] as num?)?.toInt() ?? 0,
    );

Map<String, dynamic> _$HttpListenResultToJson(HttpListenResult instance) =>
    <String, dynamic>{
      'host': instance.host,
      'port': instance.port,
    };
