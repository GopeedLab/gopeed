import 'browser_download_stub.dart'
    if (dart.library.html) 'entry/browser_download_browser.dart';

void download(String url, String name) => doDownload(url, name);
