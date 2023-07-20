import 'package:json_annotation/json_annotation.dart';

part 'request.g.dart';

@JsonSerializable(explicitToJson: true)
class Request {
  String url;
  Object? extra;

  Request({
    required this.url,
    this.extra,
  });

  factory Request.fromJson(Map<String, dynamic> json) =>
      _$RequestFromJson(json);

  Map<String, dynamic> toJson() => _$RequestToJson(this);
}

@JsonSerializable()
class ReqExtraHttp {
  String method = 'GET';
  Map<String, String> header = {};
  String body = '';

  ReqExtraHttp();

  factory ReqExtraHttp.fromJson(Map<String, dynamic> json) =>
      _$ReqExtraHttpFromJson(json);

  Map<String, dynamic> toJson() => _$ReqExtraHttpToJson(this);
}

@JsonSerializable()
class ReqExtraBt {
  List<String> trackers = [];

  ReqExtraBt();

  factory ReqExtraBt.fromJson(Map<String, dynamic> json) =>
      _$ReqExtraBtFromJson(json);

  Map<String, dynamic> toJson() => _$ReqExtraBtToJson(this);
}
