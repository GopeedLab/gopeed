// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'server_info.dart';

// **************************************************************************
// JsonSerializableGenerator
// **************************************************************************

ServerInfo _$ServerInfoFromJson(Map<String, dynamic> json) => ServerInfo(
      version: json['version'] as String,
      runtime: json['runtime'] as String,
      os: json['os'] as String,
      arch: json['arch'] as String,
      inDocker: json['inDocker'] as bool,
    );

Map<String, dynamic> _$ServerInfoToJson(ServerInfo instance) =>
    <String, dynamic>{
      'version': instance.version,
      'runtime': instance.runtime,
      'os': instance.os,
      'arch': instance.arch,
      'inDocker': instance.inDocker,
    };
