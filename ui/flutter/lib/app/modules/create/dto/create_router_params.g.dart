// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'create_router_params.dart';

// **************************************************************************
// JsonSerializableGenerator
// **************************************************************************

CreateRouterParams _$CreateRouterParamsFromJson(Map<String, dynamic> json) =>
    CreateRouterParams(
      req: json['req'] == null
          ? null
          : Request.fromJson(json['req'] as Map<String, dynamic>),
      opt: json['opt'] == null
          ? null
          : Options.fromJson(json['opt'] as Map<String, dynamic>),
    );

Map<String, dynamic> _$CreateRouterParamsToJson(CreateRouterParams instance) {
  final val = <String, dynamic>{};

  void writeNotNull(String key, dynamic value) {
    if (value != null) {
      val[key] = value;
    }
  }

  writeNotNull('req', instance.req?.toJson());
  writeNotNull('opt', instance.opt?.toJson());
  return val;
}
