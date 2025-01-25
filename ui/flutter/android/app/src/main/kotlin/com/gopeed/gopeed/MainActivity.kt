package com.gopeed.gopeed

import androidx.annotation.NonNull
import com.gopeed.libgopeed.Libgopeed
import io.flutter.embedding.android.FlutterActivity
import io.flutter.embedding.engine.FlutterEngine
import io.flutter.plugin.common.MethodChannel
import io.flutter.plugin.common.StandardMethodCodec

class MainActivity : FlutterActivity() {
    private val CHANNEL = "gopeed.com/libgopeed"

    override fun configureFlutterEngine(@NonNull flutterEngine: FlutterEngine) {
        super.configureFlutterEngine(flutterEngine)
        val taskQueue =
            flutterEngine.dartExecutor.binaryMessenger.makeBackgroundTaskQueue()
        MethodChannel(
            flutterEngine.dartExecutor.binaryMessenger,
            CHANNEL,
            StandardMethodCodec.INSTANCE,
            taskQueue
        ).setMethodCallHandler { call, result ->
            when (call.method) {
                "start" -> {
                    val cfg = call.argument<String>("cfg")
                    try {
                        val port = Libgopeed.start(cfg)
                        result.success(port)
                    } catch (e: Exception) {
                        result.error("ERROR", e.message, null)
                    }
                }
                "stop" -> {
                    Libgopeed.stop()
                    result.success(null)
                }
                else -> {
                    result.notImplemented()
                }
            }
        }
    }

}
