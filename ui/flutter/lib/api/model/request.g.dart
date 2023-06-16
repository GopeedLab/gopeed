// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'request.dart';

// **************************************************************************
// JsonSerializableGenerator
// **************************************************************************

Request _$RequestFromJson(Map<String, dynamic> json) => Request(
      url: json['url'] as String,
      extra: json['extra'] as Map<String, dynamic>?,
    );

Map<String, dynamic> _$RequestToJson(Request instance) {
  final val = <String, dynamic>{
    'url': instance.url,
  };

  void writeNotNull(String key, dynamic value) {
    if (value != null) {
      val[key] = value;
    }
  }

  writeNotNull('extra', instance.extra);
  return val;
}

ReqExtraHttp _$ReqExtraHttpFromJson(Map<String, dynamic> json) => ReqExtraHttp()
  ..method = json['method'] as String
  ..headers = Map<String, String>.from(json['headers'] as Map)
  ..body = json['body'] as String;

Map<String, dynamic> _$ReqExtraHttpToJson(ReqExtraHttp instance) =>
    <String, dynamic>{
      'method': instance.method,
      'headers': instance.headers,
      'body': instance.body,
    };
