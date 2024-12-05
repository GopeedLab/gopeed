import 'package:json_annotation/json_annotation.dart';

part 'invoke_request.g.dart';

@JsonSerializable()
class InvokeRequest {
  String method;
  List<dynamic> params;

  InvokeRequest({
    this.method = '',
    this.params = const [],
  });

  factory InvokeRequest.fromJson(Map<String, dynamic> json) =>
      _$InvokeRequestFromJson(json);

  Map<String, dynamic> toJson() => _$InvokeRequestToJson(this);
}
