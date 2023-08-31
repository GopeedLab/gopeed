// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'meta.dart';

// **************************************************************************
// JsonSerializableGenerator
// **************************************************************************

Meta _$MetaFromJson(Map<String, dynamic> json) => Meta(
      req: Request.fromJson(json['req'] as Map<String, dynamic>),
      res: Resource.fromJson(json['res'] as Map<String, dynamic>),
      opts: Options.fromJson(json['opts'] as Map<String, dynamic>),
    );

Map<String, dynamic> _$MetaToJson(Meta instance) => <String, dynamic>{
      'req': instance.req.toJson(),
      'res': instance.res.toJson(),
      'opts': instance.opts.toJson(),
    };
