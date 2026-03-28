import 'package:flutter/material.dart';
import 'package:get/get.dart';

import '../../../../api/api.dart';
import '../../../../api/model/stats.dart';
import '../../../../util/util.dart';
import '../../../views/copy_button.dart';

/// Dialog displaying file hash information (SHA256, CRC32)
class HashInfoDialog extends StatelessWidget {
  final String taskId;

  const HashInfoDialog({Key? key, required this.taskId}) : super(key: key);

  @override
  Widget build(BuildContext context) {
    final statsRx = Rxn<Stats>();

    Future.microtask(() async {
      try {
        final stats = await getTaskStats(taskId);
        statsRx.value = stats;
      } catch (e) {
        // Ignore errors - hash info is optional
      }
    });

    return AlertDialog(
      title: Row(
        children: [
          Icon(Icons.security, color: Get.theme.colorScheme.primary),
          const SizedBox(width: 8),
          Text('hashInfo'.tr),
        ],
      ),
      content: SizedBox(
        width: double.maxFinite,
        child: Obx(() {
          if (statsRx.value == null) {
            return Center(
              child: Column(
                mainAxisSize: MainAxisSize.min,
                children: [
                  const CircularProgressIndicator(),
                  const SizedBox(height: 16),
                  Text('loading'.tr),
                ],
              ),
            );
          }

          final stats = statsRx.value!;
          return Column(
            mainAxisSize: MainAxisSize.min,
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Row(
                children: [
                  Icon(Icons.folder, size: 20, color: Get.theme.disabledColor),
                  const SizedBox(width: 8),
                  Text(
                    'fileSize'.tr,
                    style: Get.textTheme.titleSmall,
                  ),
                  const Spacer(),
                  Text(
                    '${Util.fmtByte(stats.fileSize)}' +
                        (stats.expectedSize > 0
                            ? ' / ${Util.fmtByte(stats.expectedSize)}'
                            : ''),
                    style: Get.textTheme.bodyMedium,
                  ),
                ],
              ),
              const Divider(height: 24),
              Row(
                children: [
                  Icon(Icons.fingerprint,
                      size: 20, color: Get.theme.colorScheme.primary),
                  const SizedBox(width: 8),
                  Text(
                    'sha256Label'.tr,
                    style: Get.textTheme.titleSmall,
                  ),
                  const Spacer(),
                  CopyButton(stats.sha256),
                ],
              ),
              const SizedBox(height: 8),
              Container(
                padding: const EdgeInsets.all(8),
                decoration: BoxDecoration(
                  color: Get.theme.colorScheme.surfaceContainerHighest,
                  borderRadius: BorderRadius.circular(4),
                ),
                child: Row(
                  children: [
                    Expanded(
                      child: Text(
                        stats.sha256,
                        style: Get.textTheme.bodySmall?.copyWith(
                          fontFamily: 'monospace',
                          color: stats.integrityVerified
                              ? Colors.green
                              : Colors.red,
                        ),
                        overflow: TextOverflow.ellipsis,
                      ),
                    ),
                  ],
                ),
              ),
              const SizedBox(height: 16),
              Row(
                children: [
                  Icon(Icons.fingerprint,
                      size: 20, color: Get.theme.colorScheme.primary),
                  const SizedBox(width: 8),
                  Text(
                    'crc32Label'.tr,
                    style: Get.textTheme.titleSmall,
                  ),
                  const Spacer(),
                  CopyButton(stats.crc32),
                ],
              ),
              const SizedBox(height: 8),
              Container(
                padding: const EdgeInsets.all(8),
                decoration: BoxDecoration(
                  color: Get.theme.colorScheme.surfaceContainerHighest,
                  borderRadius: BorderRadius.circular(4),
                ),
                child: Row(
                  children: [
                    Expanded(
                      child: Text(
                        stats.crc32,
                        style: Get.textTheme.bodySmall?.copyWith(
                          fontFamily: 'monospace',
                          color: stats.integrityVerified
                              ? Colors.green
                              : Colors.red,
                        ),
                        overflow: TextOverflow.ellipsis,
                      ),
                    ),
                  ],
                ),
              ),
              const Divider(height: 24),
              Row(
                children: [
                  Icon(Icons.info_outline,
                      size: 20, color: Get.theme.disabledColor),
                  const SizedBox(width: 8),
                  Text(
                    'verificationStatus'.tr,
                    style: Get.textTheme.titleSmall,
                  ),
                  const Spacer(),
                  Icon(
                    stats.integrityVerified
                        ? Icons.check_circle
                        : Icons.error_outline,
                    color: stats.integrityVerified
                        ? Colors.green
                        : Colors.orange,
                    size: 20,
                  ),
                  const SizedBox(width: 4),
                  Text(
                    stats.integrityVerified ? 'verified'.tr : 'notVerified'.tr,
                    style: Get.textTheme.bodySmall?.copyWith(
                      color: stats.integrityVerified
                          ? Colors.green
                          : Colors.orange,
                      fontWeight: FontWeight.bold,
                    ),
                  ),
                ],
              ),
            ],
          );
        }),
      ),
      actions: [
        TextButton(
          onPressed: () => Get.back(),
          child: Text('cancel'.tr),
        ),
      ],
    );
  }
}
