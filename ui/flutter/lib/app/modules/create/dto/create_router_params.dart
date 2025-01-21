import 'package:json_annotation/json_annotation.dart';

import '../../../../api/model/options.dart';
import '../../../../api/model/request.dart';

part 'create_router_params.g.dart';

@JsonSerializable(explicitToJson: true)
class CreateRouterParams {
  Request? req;
  Options? opt;

  CreateRouterParams({
    this.req,
    this.opt,
  });

  factory CreateRouterParams.fromJson(
      Map<String, dynamic> json,
      ) =>
      _$CreateRouterParamsFromJson(json);

  Map<String, dynamic> toJson() => _$CreateRouterParamsToJson(this);
}
