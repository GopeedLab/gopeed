// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'stats.dart';

// **************************************************************************
// JsonSerializableGenerator
// **************************************************************************

Stats _$StatsFromJson(Map<String, dynamic> json) => Stats(
      connections: (json['connections'] as List<dynamic>)
          .map((e) => StatsConnection.fromJson(e as Map<String, dynamic>))
          .toList(),
      sha256: json['sha256'] as String,
      crc32: json['crc32'] as String,
      fileSize: json['fileSize'] as int,
      expectedSize: json['expectedSize'] as int,
      integrityVerified: json['integrityVerified'] as bool,
    );

Map<String, dynamic> _$StatsToJson(Stats instance) => <String, dynamic>{
      'connections': instance.connections,
      'sha256': instance.sha256,
      'crc32': instance.crc32,
      'fileSize': instance.fileSize,
      'expectedSize': instance.expectedSize,
      'integrityVerified': instance.integrityVerified,
    };

StatsConnection _$StatsConnectionFromJson(Map<String, dynamic> json) =>
    StatsConnection(
      downloaded: json['downloaded'] as int,
      completed: json['completed'] as bool,
      failed: json['failed'] as bool,
      retryTimes: json['retryTimes'] as int,
    );

Map<String, dynamic> _$StatsConnectionToJson(StatsConnection instance) =>
    <String, dynamic>{
      'downloaded': instance.downloaded,
      'completed': instance.completed,
      'failed': instance.failed,
      'retryTimes': instance.retryTimes,
    };
