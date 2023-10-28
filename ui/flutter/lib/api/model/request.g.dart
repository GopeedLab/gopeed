// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'request.dart';

// **************************************************************************
// JsonSerializableGenerator
// **************************************************************************

Request _$RequestFromJson(Map<String, dynamic> json) => Request(
      url: json['url'] as String,
      extra: json['extra'],
      labels: (json['labels'] as Map<String, dynamic>?)?.map(
        (k, e) => MapEntry(k, e as String),
      ),
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
  writeNotNull('labels', instance.labels);
  return val;
}

ReqExtraHttp _$ReqExtraHttpFromJson(Map<String, dynamic> json) => ReqExtraHttp()
  ..method = json['method'] as String
  ..header = Map<String, String>.from(json['header'] as Map)
  ..body = json['body'] as String;

Map<String, dynamic> _$ReqExtraHttpToJson(ReqExtraHttp instance) =>
    <String, dynamic>{
      'method': instance.method,
      'header': instance.header,
      'body': instance.body,
    };

ReqExtraBt _$ReqExtraBtFromJson(Map<String, dynamic> json) => ReqExtraBt()
  ..trackers =
      (json['trackers'] as List<dynamic>).map((e) => e as String).toList();

Map<String, dynamic> _$ReqExtraBtToJson(ReqExtraBt instance) =>
    <String, dynamic>{
      'trackers': instance.trackers,
    };
