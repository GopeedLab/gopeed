import 'dart:async';

import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../api/gopeed_site_api.dart';
import '../../../api/model/store_extension.dart';

final gopeedSiteApiProvider = Provider<GopeedSiteApi>((ref) => GopeedSiteApi.instance);

typedef StoreExtensionDetailsKey = ({String id, String version});

// A shared future deduplicates drawer/page requests. Successful details survive
// closing the view, while failures can be retried and new versions get a new key.
final storeExtensionDetailsProvider = FutureProvider.autoDispose.family<StoreExtension, StoreExtensionDetailsKey>((
  ref,
  key,
) async {
  final extension = await ref.watch(gopeedSiteApiProvider).getExtension(key.id, version: key.version);
  if (ref.mounted) {
    final keepAlive = ref.keepAlive();
    final expiry = Timer(const Duration(minutes: 30), keepAlive.close);
    ref.onDispose(expiry.cancel);
  }
  return extension;
});
