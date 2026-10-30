import 'package:json_annotation/json_annotation.dart';

part 'request.g.dart';

@JsonSerializable(explicitToJson: true)
class Request {
  String? rawUrl;
  String url;
  Object? extra;
  Map<String, String>? labels;
  RequestProxy? proxy;
  bool skipVerifyCert;

  Request({this.rawUrl, required this.url, this.extra, this.labels, this.proxy, this.skipVerifyCert = false});

  factory Request.fromJson(Map<String, dynamic> json) => _$RequestFromJson(json);

  Map<String, dynamic> toJson() => _$RequestToJson(this);
}

@JsonSerializable()
class ReqExtraHttp {
  String method;
  Map<String, String> header;
  String body;

  ReqExtraHttp({this.method = 'GET', this.header = const {}, this.body = ''});

  factory ReqExtraHttp.fromJson(Map<String, dynamic> json) => _$ReqExtraHttpFromJson(json);

  Map<String, dynamic> toJson() => _$ReqExtraHttpToJson(this);
}

@JsonSerializable()
class ReqExtraBt {
  List<String> trackers;

  ReqExtraBt({this.trackers = const []});

  factory ReqExtraBt.fromJson(Map<String, dynamic> json) => _$ReqExtraBtFromJson(json);

  Map<String, dynamic> toJson() => _$ReqExtraBtToJson(this);
}

/// The request extra of an FTP task. It mirrors `pkg/protocol/ftp.ReqExtra`;
/// [tls] only applies to `ftp://` URLs, because `ftps://` always uses
/// implicit TLS and `ftpes://` always uses explicit TLS.
@JsonSerializable()
class ReqExtraFtp {
  String tls;

  ReqExtraFtp({this.tls = 'none'});

  factory ReqExtraFtp.fromJson(Map<String, dynamic> json) => _$ReqExtraFtpFromJson(json);

  Map<String, dynamic> toJson() => _$ReqExtraFtpToJson(this);
}

enum RequestProxyMode { follow, none, custom }

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

  factory RequestProxy.fromJson(Map<String, dynamic> json) => _$RequestProxyFromJson(json);

  Map<String, dynamic> toJson() => _$RequestProxyToJson(this);
}
