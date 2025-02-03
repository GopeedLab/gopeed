import 'package:flutter/material.dart';

import '../../icon/gopeed_icons.dart';

final Map<IconData, List<String>> iconConfigMap = {
  Gopeed.install: ['exe', 'msi', 'dmg', 'deb', 'rpm'],
  Gopeed.android: ['apk'],
  Gopeed.app_store_ios: ['ipa'],
  Gopeed.file_bt: ['torrent'],
  Gopeed.cd: ['iso'],
  Gopeed.html5: ['html', 'htm'],
  Gopeed.file_alt: ['txt', 'md', 'log', 'csv', 'tsv', 'json', 'yaml', 'yml'],
  Gopeed.file_pdf: ['pdf'],
  Gopeed.file_word: ['doc', 'docx'],
  Gopeed.file_excel: ['xls', 'xlsx'],
  Gopeed.file_powerpoint: ['ppt', 'pptx'],
  Gopeed.file_archive: ['zip', 'rar', '7z', 'tar', 'gz', 'bz2', 'xz'],
  Gopeed.file_image: [
    'jpg',
    'jpeg',
    'png',
    'gif',
    'bmp',
    'tiff',
    'svg',
    'webp'
  ],
  Gopeed.file_audio: ['mp3', 'wav', 'flac', 'aac', 'ogg', 'wma', 'm4a'],
  Gopeed.file_video: ['mp4', 'avi', 'mkv', 'mov', 'wmv', 'flv', 'webm'],
  Gopeed.file_code: [
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
  ],
  Gopeed.file: [''],
};

final Map<String, IconData> _iconCache = Map.fromEntries(
  iconConfigMap.entries.expand(
    (entry) => entry.value.map((ext) => MapEntry(ext, entry.key)),
  ),
);

String fileExt(String? name) {
  if (name == null) {
    return '';
  }

  final ext = name.split('.').last;
  if (ext.length > 8) {
    return '';
  }

  return ext.toLowerCase();
}

IconData fileIcon(String? name,
    {bool isFolder = false, bool isBitTorrent = false}) {
  if (isFolder) {
    return isBitTorrent ? Gopeed.folder_bt : Gopeed.folder;
  }

  final ext = fileExt(name);
  return _iconCache[ext] ?? Gopeed.file;
}
