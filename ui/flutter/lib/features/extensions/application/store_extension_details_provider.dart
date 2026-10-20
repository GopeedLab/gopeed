import 'dart:async';

import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../../api/gopeed_site_api.dart';
import '../../../api/model/store_extension.dart';

final gopeedSiteApiProvider = Provider<GopeedSiteApi>((ref) => GopeedSiteApi.instance);

typedef StoreExtensionDetailsKey = ({String id, String version});

// A shared future deduplicates drawer/page requests. Successful details survive
// closing the view, while failures can be retried and new versions get a new key.
final storeExtensionDetailsProvider = FutureProvider.autoDispose.family<StoreExtension, StoreExtensionDetailsKey>(
  (ref, key) async {
    final api = ref.watch(gopeedSiteApiProvider);
    // Keep the whole three-attempt cycle loading; expose only the final error.
    for (var attempt = 0; ; attempt++) {
      try {
        final extension = await api.getExtension(key.id, version: key.version);
        if (ref.mounted) {
          final keepAlive = ref.keepAlive();
          final expiry = Timer(const Duration(minutes: 30), keepAlive.close);
          ref.onDispose(expiry.cancel);
        }
        return extension;
      } catch (_) {
        if (attempt == 2 || !ref.mounted) rethrow;
        await Future<void>.delayed(Duration(milliseconds: 300 * (attempt + 1)));
        if (!ref.mounted) rethrow;
      }
    }
  },
  // The bounded cycle above owns retries, including after a manual reload.
  retry: (retryCount, error) => null,
);
