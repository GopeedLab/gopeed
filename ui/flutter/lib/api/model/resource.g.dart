// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'resource.dart';

// **************************************************************************
// JsonSerializableGenerator
// **************************************************************************

Resource _$ResourceFromJson(Map<String, dynamic> json) => Resource(
      name: json['name'] as String,
      size: json['size'] as int,
      range: json['range'] as bool,
      files: (json['files'] as List<dynamic>)
          .map((e) => FileInfo.fromJson(e as Map<String, dynamic>))
          .toList(),
      hash: json['hash'] as String,
    );

Map<String, dynamic> _$ResourceToJson(Resource instance) => <String, dynamic>{
      'name': instance.name,
      'size': instance.size,
      'range': instance.range,
      'files': instance.files.map((e) => e.toJson()).toList(),
      'hash': instance.hash,
    };

FileInfo _$FileInfoFromJson(Map<String, dynamic> json) => FileInfo(
      name: json['name'] as String,
      path: json['path'] as String,
      size: json['size'] as int,
    );

Map<String, dynamic> _$FileInfoToJson(FileInfo instance) => <String, dynamic>{
      'name': instance.name,
      'path': instance.path,
      'size': instance.size,
    };
