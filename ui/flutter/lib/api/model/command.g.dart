// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'command.dart';

// **************************************************************************
// JsonSerializableGenerator
// **************************************************************************

Command _$CommandFromJson(Map<String, dynamic> json) => Command(
      protocol: json['protocol'] as String,
      action: json['action'] as String,
      params: json['params'],
    );

Map<String, dynamic> _$CommandToJson(Command instance) {
  final val = <String, dynamic>{
    'protocol': instance.protocol,
    'action': instance.action,
  };

  void writeNotNull(String key, dynamic value) {
    if (value != null) {
      val[key] = value;
    }
  }

  writeNotNull('params', instance.params);
  return val;
}
