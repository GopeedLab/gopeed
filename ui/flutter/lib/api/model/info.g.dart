// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'info.dart';

// **************************************************************************
// JsonSerializableGenerator
// **************************************************************************

Info _$InfoFromJson(Map<String, dynamic> json) => Info(
      version: json['version'] as String? ?? '',
      runtime: json['runtime'] as String? ?? '',
      os: json['os'] as String? ?? '',
      arch: json['arch'] as String? ?? '',
      inDocker: json['inDocker'] as bool? ?? false,
    );

Map<String, dynamic> _$InfoToJson(Info instance) => <String, dynamic>{
      'version': instance.version,
      'runtime': instance.runtime,
      'os': instance.os,
      'arch': instance.arch,
      'inDocker': instance.inDocker,
    };
