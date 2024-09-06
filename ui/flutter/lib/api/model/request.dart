import 'package:json_annotation/json_annotation.dart';

part 'request.g.dart';

@JsonSerializable(explicitToJson: true)
class Request {
  String url;
  Object? extra;
  Map<String, String>? labels = {};
  RequestProxy? proxy;

  Request({
    required this.url,
    this.extra,
    this.labels,
    this.proxy,
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

enum RequestProxyMode {
  follow,
  none,
  custom,
}

@JsonSerializable()
class RequestProxy {
  RequestProxyMode mode = RequestProxyMode.follow;
  String scheme = 'http';
  String host = '';
  String usr = '';
  String pwd = '';

  RequestProxy();

  factory RequestProxy.fromJson(Map<String, dynamic> json) =>
      _$RequestProxyFromJson(json);
  Map<String, dynamic> toJson() => _$RequestProxyToJson(this);
}
