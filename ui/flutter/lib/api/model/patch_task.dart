import 'options.dart';
import 'request.dart';

class PatchTask {
  final Request? req;
  final Options? opts;

  PatchTask({this.req, this.opts});

  factory PatchTask.fromJson(Map<String, dynamic> json) => PatchTask(
    req: json['req'] == null ? null : Request.fromJson(json['req'] as Map<String, dynamic>),
    opts: json['opts'] == null ? null : Options.fromJson(json['opts'] as Map<String, dynamic>),
  );

  Map<String, dynamic> toJson() => {'req': req?.toJson(), 'opts': opts?.toJson()};
}
