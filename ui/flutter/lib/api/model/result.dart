import 'package:json_annotation/json_annotation.dart';

part 'result.g.dart';

@JsonSerializable(genericArgumentFactories: true)
class Result<T> {
  String? msg;
  T? data;

  Result({
    this.msg,
    this.data,
  });

  factory Result.fromJson(
    Map<String, dynamic> json,
    T Function(dynamic json) fromJsonT,
  ) =>
      _$ResultFromJson(json, fromJsonT);
  Map<String, dynamic> toJson() => {
        'msg': msg,
        'data': data is List
            ? (data as dynamic)?.map((e) => e.toJson()).toList()
            : (data as dynamic)?.toJson(),
      };
}
