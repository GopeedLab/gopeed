import 'package:json_annotation/json_annotation.dart';

import 'resource.dart';

part 'resolve_result.g.dart';

@JsonSerializable(explicitToJson: true)
class ResolveResult {
  String id;
  Resource res;

  /// The category directory the created task would be routed to. Empty when
  /// auto categorize does not apply. It is a hint for the ui, the create call
  /// decides the final path.
  String suggestedDownloadDir;

  ResolveResult({this.id = "", required this.res, this.suggestedDownloadDir = ""});

  factory ResolveResult.fromJson(Map<String, dynamic> json) => _$ResolveResultFromJson(json);
  Map<String, dynamic> toJson() => _$ResolveResultToJson(this);
}
