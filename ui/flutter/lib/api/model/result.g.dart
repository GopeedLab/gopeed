// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'result.dart';

// **************************************************************************
// JsonSerializableGenerator
// **************************************************************************

Result<T> _$ResultFromJson<T>(Map<String, dynamic> json, T Function(Object? json) fromJsonT) => Result<T>(
  code: (json['code'] as num).toInt(),
  msg: json['msg'] as String?,
  data: _$nullableGenericFromJson(json['data'], fromJsonT),
);

T? _$nullableGenericFromJson<T>(Object? input, T Function(Object? json) fromJson) =>
    input == null ? null : fromJson(input);
