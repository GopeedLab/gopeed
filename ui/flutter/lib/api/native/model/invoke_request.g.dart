// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'invoke_request.dart';

// **************************************************************************
// JsonSerializableGenerator
// **************************************************************************

InvokeRequest _$InvokeRequestFromJson(Map<String, dynamic> json) =>
    InvokeRequest(
      method: json['method'] as String? ?? '',
      params: (json['params'] as List<dynamic>?)
              ?.map((e) => e as String)
              .toList() ??
          const [],
    );

Map<String, dynamic> _$InvokeRequestToJson(InvokeRequest instance) =>
    <String, dynamic>{
      'method': instance.method,
      'params': instance.params,
    };
