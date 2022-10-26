// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'resource.dart';

// **************************************************************************
// JsonSerializableGenerator
// **************************************************************************

Resource _$ResourceFromJson(Map<String, dynamic> json) => Resource(
      req: Request.fromJson(json['req'] as Map<String, dynamic>),
      name: json['name'] as String,
      size: json['size'] as int,
      range: json['range'] as bool,
      files: (json['files'] as List<dynamic>)
          .map((e) => FileInfo.fromJson(e as Map<String, dynamic>))
          .toList(),
      extra: json['extra'] as Map<String, dynamic>?,
    );

Map<String, dynamic> _$ResourceToJson(Resource instance) {
  final val = <String, dynamic>{
    'req': instance.req.toJson(),
    'name': instance.name,
    'size': instance.size,
    'range': instance.range,
    'files': instance.files.map((e) => e.toJson()).toList(),
  };

  void writeNotNull(String key, dynamic value) {
    if (value != null) {
      val[key] = value;
    }
  }

  writeNotNull('extra', instance.extra);
  return val;
}

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
