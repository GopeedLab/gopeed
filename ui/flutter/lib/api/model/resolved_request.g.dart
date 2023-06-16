// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'resolved_request.dart';

// **************************************************************************
// JsonSerializableGenerator
// **************************************************************************

ResolvedRequest _$ResolvedRequestFromJson(Map<String, dynamic> json) =>
    ResolvedRequest(
      url: json['url'] as String,
      extra: json['extra'] as Map<String, dynamic>?,
      res: json['res'] == null
          ? null
          : Resource.fromJson(json['res'] as Map<String, dynamic>),
    );

Map<String, dynamic> _$ResolvedRequestToJson(ResolvedRequest instance) {
  final val = <String, dynamic>{
    'url': instance.url,
  };

  void writeNotNull(String key, dynamic value) {
    if (value != null) {
      val[key] = value;
    }
  }

  writeNotNull('extra', instance.extra);
  writeNotNull('res', instance.res?.toJson());
  return val;
}
