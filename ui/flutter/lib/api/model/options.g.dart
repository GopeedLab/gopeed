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
      autoTorrent: json['autoTorrent'] as bool?,
      deleteTorrentAfterDownload: json['deleteTorrentAfterDownload'] as bool?,
      autoExtract: json['autoExtract'] as bool?,
      archivePassword: json['archivePassword'] as String? ?? '',
      deleteAfterExtract: json['deleteAfterExtract'] as bool? ?? false,
    );

Map<String, dynamic> _$OptsExtraHttpToJson(OptsExtraHttp instance) {
  final val = <String, dynamic>{
    'connections': instance.connections,
  };

  void writeNotNull(String key, dynamic value) {
    if (value != null) {
      val[key] = value;
    }
  }

  writeNotNull('autoTorrent', instance.autoTorrent);
  writeNotNull(
      'deleteTorrentAfterDownload', instance.deleteTorrentAfterDownload);
  writeNotNull('autoExtract', instance.autoExtract);
  val['archivePassword'] = instance.archivePassword;
  val['deleteAfterExtract'] = instance.deleteAfterExtract;
  return val;
}
