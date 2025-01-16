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
      proxy: json['proxy'] == null
          ? null
          : RequestProxy.fromJson(json['proxy'] as Map<String, dynamic>),
      skipVerifyCert: json['skipVerifyCert'] as bool? ?? false,
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
  writeNotNull('proxy', instance.proxy?.toJson());
  val['skipVerifyCert'] = instance.skipVerifyCert;
  return val;
}

ReqExtraHttp _$ReqExtraHttpFromJson(Map<String, dynamic> json) => ReqExtraHttp(
      method: json['method'] as String? ?? 'GET',
      header: (json['header'] as Map<String, dynamic>?)?.map(
            (k, e) => MapEntry(k, e as String),
          ) ??
          const {},
      body: json['body'] as String? ?? '',
    );

Map<String, dynamic> _$ReqExtraHttpToJson(ReqExtraHttp instance) =>
    <String, dynamic>{
      'method': instance.method,
      'header': instance.header,
      'body': instance.body,
    };

ReqExtraBt _$ReqExtraBtFromJson(Map<String, dynamic> json) => ReqExtraBt(
      trackers: (json['trackers'] as List<dynamic>?)
              ?.map((e) => e as String)
              .toList() ??
          const [],
    );

Map<String, dynamic> _$ReqExtraBtToJson(ReqExtraBt instance) =>
    <String, dynamic>{
      'trackers': instance.trackers,
    };

RequestProxy _$RequestProxyFromJson(Map<String, dynamic> json) => RequestProxy(
      mode: $enumDecodeNullable(_$RequestProxyModeEnumMap, json['mode']) ??
          RequestProxyMode.follow,
      scheme: json['scheme'] as String? ?? 'http',
      host: json['host'] as String? ?? '',
      usr: json['usr'] as String? ?? '',
      pwd: json['pwd'] as String? ?? '',
    );

Map<String, dynamic> _$RequestProxyToJson(RequestProxy instance) =>
    <String, dynamic>{
      'mode': _$RequestProxyModeEnumMap[instance.mode]!,
      'scheme': instance.scheme,
      'host': instance.host,
      'usr': instance.usr,
      'pwd': instance.pwd,
    };

const _$RequestProxyModeEnumMap = {
  RequestProxyMode.follow: 'follow',
  RequestProxyMode.none: 'none',
  RequestProxyMode.custom: 'custom',
};
