import 'package:dio/dio.dart';

// List of mirrors for GitHub
const _sourceMirror = [
  // github.tbedu.top: https://github.tbedu.top/https://raw.githubusercontent.com/ngosang/trackerslist/master/trackers_all.txt
  ["https://github.tbedu.top/", r"(.*)"],
  // fastgit.cc: https://fastgit.cc/https://raw.githubusercontent.com/ngosang/trackerslist/master/trackers_all.txt
  ["https://fastgit.cc/", r"(.*)"],
  // gitproxy.click: https://gitproxy.click/https://raw.githubusercontent.com/ngosang/trackerslist/master/trackers_all.txt
  ["https://gitproxy.click/", r"(.*)"],
  // jsdelivr: https://fastly.jsdelivr.net/gh/ngosang/trackerslist/trackers_all.txt
  [
    "https://fastly.jsdelivr.net/gh",
    r".*raw.githubusercontent.com(/.*)(/.*)/(?:master|main)(/.*)"
  ],
];

// List of mirrors for GitHub release
const _assertMirror = [
  // github.tbedu.top: https://https://github.tbedu.top/https://github.com/GopeedLab/gopeed/releases/download/v1.6.10/Gopeed-v1.6.10-windows-amd64.zip
  ["https://github.tbedu.top/", r"(.*)"],
  // fastgit.cc: https://fastgit.cc/https://github.com/GopeedLab/gopeed/releases/download/v1.6.10/Gopeed-v1.6.10-windows-amd64.zip
  ["https://fastgit.cc/", r"(.*)"],
  // gitproxy.click: https://gitproxy.click/https://github.com/GopeedLab/gopeed/releases/download/v1.6.10/Gopeed-v1.6.10-windows-amd64.zip
  ["https://gitproxy.click/", r"(.*)"],
];

enum MirrorType {
  githubSource,
  githubRelease,
}

/// Auto detect the best mirror for the given [rawUrl] and [type]
Future<String> githubAutoMirror(String rawUrl, MirrorType type) async {
  final mirrorUrls = githubMirrorUrls(rawUrl, type);

  // Ping all mirrors and get the fastest one
  final pingResult = await Future.wait(mirrorUrls.map((e) async {
    final client = Dio()
      ..options.sendTimeout = const Duration(seconds: 3)
      ..options.connectTimeout = const Duration(seconds: 3);
    var time = DateTime.now().millisecondsSinceEpoch;
    try {
      final response = await client.head(e);
      if (response.statusCode == 200) {
        time = DateTime.now().millisecondsSinceEpoch - time;
        return (e, time);
      }
    } catch (e) {
      // ignore
    } finally {
      client.close();
    }
    return (e, -1);
  }));

  var list = pingResult.where((e) => e.$2 != -1).toList()
    ..sort((a, b) => a.$2.compareTo(b.$2));
  if (list.isNotEmpty) {
    return list.first.$1;
  }

  return rawUrl;
}

List<String> githubMirrorUrls(String rawUrl, MirrorType type) {
  final mirrors = switch (type) {
    MirrorType.githubSource => _sourceMirror,
    MirrorType.githubRelease => _assertMirror,
  };

  final ret = <String>[];
  for (final cdn in mirrors) {
    final reg = RegExp(cdn[1]);
    final match = reg.firstMatch(rawUrl.toString());
    var matchStr = "";
    for (var i = 1; i <= match!.groupCount; i++) {
      matchStr += match.group(i)!;
    }
    ret.add("${cdn[0]}$matchStr");
  }
  return ret;
}
