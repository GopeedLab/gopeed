// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'options.dart';

// **************************************************************************
// JsonSerializableGenerator
// **************************************************************************

Options _$OptionsFromJson(Map<String, dynamic> json) => Options(
      name: json['name'] as String,
      path: json['path'] as String,
      selectFiles:
          (json['selectFiles'] as List<dynamic>).map((e) => e as int).toList(),
    );

Map<String, dynamic> _$OptionsToJson(Options instance) => <String, dynamic>{
      'name': instance.name,
      'path': instance.path,
      'selectFiles': instance.selectFiles,
    };
