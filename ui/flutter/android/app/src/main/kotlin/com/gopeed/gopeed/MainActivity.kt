package com.gopeed.gopeed

import com.gopeed.libgopeed.Libgopeed
import io.flutter.embedding.android.FlutterActivity
import io.flutter.embedding.engine.FlutterEngine
import io.flutter.plugin.common.MethodChannel
import io.flutter.plugin.common.StandardMethodCodec

class MainActivity : FlutterActivity() {
    override fun configureFlutterEngine(flutterEngine: FlutterEngine) {
        super.configureFlutterEngine(flutterEngine)

        val taskQueue =
            flutterEngine.dartExecutor.binaryMessenger.makeBackgroundTaskQueue()
        MethodChannel(
            flutterEngine.dartExecutor.binaryMessenger,
            LIBGOPEED_CHANNEL,
            StandardMethodCodec.INSTANCE,
            taskQueue,
        ).setMethodCallHandler { call, result ->
            when (call.method) {
                "start" -> {
                    try {
                        result.success(Libgopeed.start(call.argument<String>("cfg")))
                    } catch (error: Exception) {
                        result.error("ERROR", error.message, null)
                    }
                }
                "stop" -> {
                    Libgopeed.stop()
                    result.success(null)
                }
                else -> result.notImplemented()
            }
        }
    }

    private companion object {
        const val LIBGOPEED_CHANNEL = "gopeed.com/libgopeed"
    }
}
