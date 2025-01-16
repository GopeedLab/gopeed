import 'package:json_annotation/json_annotation.dart';

part 'request.g.dart';

@JsonSerializable(explicitToJson: true)
class Request {
  String url;
  Object? extra;
  Map<String, String>? labels;
  RequestProxy? proxy;
  bool skipVerifyCert;

  Request({
    required this.url,
    this.extra,
    this.labels,
    this.proxy,
    this.skipVerifyCert = false,
  });

  factory Request.fromJson(Map<String, dynamic> json) =>
      _$RequestFromJson(json);

  Map<String, dynamic> toJson() => _$RequestToJson(this);
}

@JsonSerializable()
class ReqExtraHttp {
  String method;
  Map<String, String> header;
  String body;

  ReqExtraHttp({
    this.method = 'GET',
    this.header = const {},
    this.body = '',
  });

  factory ReqExtraHttp.fromJson(Map<String, dynamic> json) =>
      _$ReqExtraHttpFromJson(json);

  Map<String, dynamic> toJson() => _$ReqExtraHttpToJson(this);
}

@JsonSerializable()
class ReqExtraBt {
  List<String> trackers;

  ReqExtraBt({
    this.trackers = const [],
  });

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
  RequestProxyMode mode;
  String scheme;
  String host;
  String usr;
  String pwd;

  RequestProxy({
    this.mode = RequestProxyMode.follow,
    this.scheme = 'http',
    this.host = '',
    this.usr = '',
    this.pwd = '',
  });

  factory RequestProxy.fromJson(Map<String, dynamic> json) =>
      _$RequestProxyFromJson(json);

  Map<String, dynamic> toJson() => _$RequestProxyToJson(this);
}
