// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'entity.dart';

// **************************************************************************
// JsonSerializableGenerator
// **************************************************************************

StartConfigEntity _$StartConfigEntityFromJson(Map<String, dynamic> json) =>
    StartConfigEntity(
      network: json['network'] as String,
      address: json['address'] as String,
      apiToken: json['apiToken'] as String,
    );

Map<String, dynamic> _$StartConfigEntityToJson(StartConfigEntity instance) =>
    <String, dynamic>{
      'network': instance.network,
      'address': instance.address,
      'apiToken': instance.apiToken,
    };

WindowStateEntity _$WindowStateEntityFromJson(Map<String, dynamic> json) =>
    WindowStateEntity(
      isMaximized: json['isMaximized'] as bool?,
      width: (json['width'] as num?)?.toDouble(),
      height: (json['height'] as num?)?.toDouble(),
    );

Map<String, dynamic> _$WindowStateEntityToJson(WindowStateEntity instance) {
  final val = <String, dynamic>{};

  void writeNotNull(String key, dynamic value) {
    if (value != null) {
      val[key] = value;
    }
  }

  writeNotNull('isMaximized', instance.isMaximized);
  writeNotNull('width', instance.width);
  writeNotNull('height', instance.height);
  return val;
}
