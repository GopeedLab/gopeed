import 'options.dart';
import 'request.dart';
import 'resource.dart';
import 'package:json_annotation/json_annotation.dart';

part 'meta.g.dart';

@JsonSerializable(explicitToJson: true)
class Meta {
  @JsonKey(fromJson: _requestFromJson)
  Request req;
  Resource? res;
  Options opts;

  Meta({required this.req, required this.opts});

  factory Meta.fromJson(Map<String, dynamic> json) => _$MetaFromJson(json);
  Map<String, dynamic> toJson() => _$MetaToJson(this);
}

Request _requestFromJson(Map<String, dynamic>? json) => json == null ? Request(url: '') : Request.fromJson(json);
