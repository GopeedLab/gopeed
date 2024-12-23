import 'package:flutter/material.dart';

import '../../icon/gopeed_icons.dart';

final Map<List<String>, IconData> _iconConfigMap = {
  ['exe', 'msi', 'dmg', 'deb', 'rpm']: Gopeed.install,
  ['apk']: Gopeed.android,
  ['ipa']: Gopeed.app_store_ios,
  ['html', 'htm']: Gopeed.html5,
  ['iso']: Gopeed.cd,
  ['pdf']: Gopeed.file_pdf,
  ['doc', 'docx']: Gopeed.file_word,
  ['xls', 'xlsx']: Gopeed.file_excel,
  ['ppt', 'pptx']: Gopeed.file_powerpoint,
  ['zip', 'rar', '7z', 'tar', 'gz', 'bz2', 'xz']: Gopeed.file_archive,
  ['jpg', 'jpeg', 'png', 'gif', 'bmp', 'tiff', 'svg', 'webp']:
      Gopeed.file_image,
  ['mp3', 'wav', 'flac', 'aac', 'ogg', 'wma', 'm4a']: Gopeed.file_audio,
  ['mp4', 'avi', 'mkv', 'mov', 'wmv', 'flv', 'webm']: Gopeed.file_video,
  [
    'js',
    'css',
    'json',
    'xml',
    'java',
    'cpp',
    'dart',
    'py',
    'rb',
    'php',
    'ts',
    'swift',
    'go',
    'rs'
  ]: Gopeed.file_code,
  ['']: Gopeed.file_download,
};

final Map<String, IconData> _iconCache =
    _iconConfigMap.entries.fold<Map<String, IconData>>({}, (prev, entry) {
  final icon = entry.value;
  for (final ext in entry.key) {
    prev[ext] = icon;
  }
  return prev;
});

const folderIcon = Gopeed.folder;

IconData fileIcon(String? name) {
  if (name == null) {
    return _iconCache['']!;
  }

  final ext = name.split('.').last;
  if (ext.length > 8) {
    return _iconCache['']!;
  }

  final lowerExt = ext.toLowerCase();
  for (final entry in _iconCache.entries) {
    if (entry.key == lowerExt) {
      return entry.value;
    }
  }
  return _iconCache['']!;
}
