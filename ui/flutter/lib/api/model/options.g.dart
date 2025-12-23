// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'options.dart';

// **************************************************************************
// JsonSerializableGenerator
// **************************************************************************

Options _$OptionsFromJson(Map<String, dynamic> json) => Options(
      name: json['name'] as String? ?? '',
      path: json['path'] as String? ?? '',
      selectFiles: (json['selectFiles'] as List<dynamic>?)
              ?.map((e) => (e as num).toInt())
              .toList() ??
          const [],
      extra: json['extra'],
    );

Map<String, dynamic> _$OptionsToJson(Options instance) {
  final val = <String, dynamic>{
    'name': instance.name,
    'path': instance.path,
    'selectFiles': instance.selectFiles,
  };

  void writeNotNull(String key, dynamic value) {
    if (value != null) {
      val[key] = value;
    }
  }

  writeNotNull('extra', instance.extra);
  return val;
}

OptsExtraHttp _$OptsExtraHttpFromJson(Map<String, dynamic> json) =>
    OptsExtraHttp(
      connections: (json['connections'] as num?)?.toInt() ?? 0,
      autoTorrent: json['autoTorrent'] as bool? ?? false,
      autoExtract: json['autoExtract'] as bool? ?? false,
      archivePassword: json['archivePassword'] as String? ?? '',
    );

Map<String, dynamic> _$OptsExtraHttpToJson(OptsExtraHttp instance) =>
    <String, dynamic>{
      'connections': instance.connections,
      'autoTorrent': instance.autoTorrent,
      'autoExtract': instance.autoExtract,
      'archivePassword': instance.archivePassword,
    };
