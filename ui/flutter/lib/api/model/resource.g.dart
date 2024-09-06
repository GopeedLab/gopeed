// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'resource.dart';

// **************************************************************************
// JsonSerializableGenerator
// **************************************************************************

Resource _$ResourceFromJson(Map<String, dynamic> json) => Resource(
      name: json['name'] as String? ?? "",
      size: (json['size'] as num?)?.toInt() ?? 0,
      range: json['range'] as bool? ?? false,
      files: (json['files'] as List<dynamic>)
          .map((e) => FileInfo.fromJson(e as Map<String, dynamic>))
          .toList(),
      hash: json['hash'] as String? ?? "",
    );

Map<String, dynamic> _$ResourceToJson(Resource instance) => <String, dynamic>{
      'name': instance.name,
      'size': instance.size,
      'range': instance.range,
      'files': instance.files.map((e) => e.toJson()).toList(),
      'hash': instance.hash,
    };

FileInfo _$FileInfoFromJson(Map<String, dynamic> json) => FileInfo(
      path: json['path'] as String? ?? "",
      name: json['name'] as String,
      size: (json['size'] as num?)?.toInt() ?? 0,
      req: json['req'] == null
          ? null
          : Request.fromJson(json['req'] as Map<String, dynamic>),
    );

Map<String, dynamic> _$FileInfoToJson(FileInfo instance) {
  final val = <String, dynamic>{
    'path': instance.path,
    'name': instance.name,
    'size': instance.size,
  };

  void writeNotNull(String key, dynamic value) {
    if (value != null) {
      val[key] = value;
    }
  }

  writeNotNull('req', instance.req?.toJson());
  return val;
}
