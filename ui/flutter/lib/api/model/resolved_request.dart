import 'package:gopeed/api/model/request.dart';
import 'package:gopeed/api/model/resource.dart';
import 'package:json_annotation/json_annotation.dart';

part 'resolved_request.g.dart';

@JsonSerializable(explicitToJson: true)
class ResolvedRequest extends Request {
  Resource? res;

  ResolvedRequest({
    required super.url,
    super.extra,
    this.res,
  });

  factory ResolvedRequest.fromJson(Map<String, dynamic> json) =>
      _$ResolvedRequestFromJson(json);
  Map<String, dynamic> toJson() => _$ResolvedRequestToJson(this);
}
