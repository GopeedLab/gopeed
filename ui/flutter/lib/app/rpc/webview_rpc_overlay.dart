import 'package:flutter/material.dart';

import 'webview_rpc_service.dart';

class WebViewRpcOverlay extends StatefulWidget {
  const WebViewRpcOverlay({super.key});

  @override
  State<WebViewRpcOverlay> createState() => _WebViewRpcOverlayState();
}

class _WebViewRpcOverlayState extends State<WebViewRpcOverlay> {
  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addPostFrameCallback((_) {
      WebViewRpcService.instance.markOverlayReady();
    });
  }

  @override
  Widget build(BuildContext context) {
    return ValueListenableBuilder<List<WebViewRpcPageSession>>(
      valueListenable: WebViewRpcService.instance.pages,
      builder: (context, pages, _) {
        if (pages.isEmpty) {
          return const SizedBox.shrink();
        }
        return IgnorePointer(
          ignoring: false,
          child: Stack(
            children: pages
                .map((page) => _WebViewRpcPageView(page: page))
                .toList(growable: false),
          ),
        );
      },
    );
  }
}

class _WebViewRpcPageView extends StatefulWidget {
  const _WebViewRpcPageView({required this.page});

  final WebViewRpcPageSession page;

  @override
  State<_WebViewRpcPageView> createState() => _WebViewRpcPageViewState();
}

class _WebViewRpcPageViewState extends State<_WebViewRpcPageView> {
  @override
  Widget build(BuildContext context) {
    final page = widget.page;
    final content = page.buildWebView();

    return Positioned.fill(
      child: Material(
        color: Colors.black54,
        child: Center(
          child: Container(
            width: page.width > 0 ? page.width.toDouble() : 960,
            height: page.height > 0 ? page.height.toDouble() : 720,
            clipBehavior: Clip.antiAlias,
            decoration: BoxDecoration(
              color: Theme.of(context).colorScheme.surface,
              borderRadius: BorderRadius.circular(12),
              boxShadow: const [
                BoxShadow(
                  blurRadius: 24,
                  color: Colors.black26,
                ),
              ],
            ),
            child: Column(
              children: [
                Container(
                  height: 44,
                  padding: const EdgeInsets.symmetric(horizontal: 16),
                  alignment: Alignment.centerLeft,
                  color: Theme.of(context).colorScheme.surfaceContainerHighest,
                  child: Text(
                    page.title.isNotEmpty ? page.title : 'WebView',
                    maxLines: 1,
                    overflow: TextOverflow.ellipsis,
                    style: Theme.of(context).textTheme.titleSmall,
                  ),
                ),
                Expanded(child: content),
              ],
            ),
          ),
        ),
      ),
    );
  }
}
