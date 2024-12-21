// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'start_config.dart';

// **************************************************************************
// JsonSerializableGenerator
// **************************************************************************

StartConfig _$StartConfigFromJson(Map<String, dynamic> json) => StartConfig(
      storage: json['storage'] as String? ?? '',
      storageDir: json['storageDir'] as String? ?? '',
    );

Map<String, dynamic> _$StartConfigToJson(StartConfig instance) =>
    <String, dynamic>{
      'storage': instance.storage,
      'storageDir': instance.storageDir,
    };
