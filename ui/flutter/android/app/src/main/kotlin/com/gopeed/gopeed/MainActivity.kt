package com.gopeed.gopeed

import com.gopeed.libgopeed.Libgopeed
import com.gopeed.libgopeed.InvokeResultListener
import com.gopeed.libgopeed.TaskEventListener
import io.flutter.embedding.android.FlutterActivity
import io.flutter.embedding.engine.FlutterEngine
import io.flutter.plugin.common.MethodChannel
import io.flutter.plugin.common.StandardMethodCodec

class MainActivity : FlutterActivity() {
    override fun configureFlutterEngine(flutterEngine: FlutterEngine) {
        super.configureFlutterEngine(flutterEngine)

        val taskQueue =
            flutterEngine.dartExecutor.binaryMessenger.makeBackgroundTaskQueue()
        val libgopeedChannel = MethodChannel(
            flutterEngine.dartExecutor.binaryMessenger,
            LIBGOPEED_CHANNEL,
            StandardMethodCodec.INSTANCE,
            taskQueue,
        )
        libgopeedChannel.setMethodCallHandler { call, result ->
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
                "getApiServerState" -> result.success(Libgopeed.getAPIServerState())
                "startApiServer" -> result.success(Libgopeed.startAPIServer())
                "stopApiServer" -> result.success(Libgopeed.stopAPIServer())
                "restartApiServer" -> result.success(Libgopeed.restartAPIServer())
                "invoke" -> {
                    try {
                        Libgopeed.invokeAsync(
                            call.argument<String>("method") ?: "",
                            call.argument<String>("path") ?: "",
                            call.argument<String>("query") ?: "",
                            call.argument<String>("body") ?: "",
                            call.argument<Number>("requestID")?.toLong() ?: 0L,
                            object : InvokeResultListener {
                                override fun onResult(requestID: Long, success: Boolean, payload: String?) {
                                    runOnUiThread {
                                        if (success) {
                                            result.success(payload ?: "")
                                        } else {
                                            result.error("ERROR", payload ?: "InvokeAsync failed", null)
                                        }
                                    }
                                }
                            },
                        )
                    } catch (error: Exception) {
                        result.error("ERROR", error.message, null)
                    }
                }
                "subscribeTaskEvents" -> {
                    val mask = call.argument<Number>("mask")?.toLong() ?: 0L
                    if (mask == 0L) {
                        Libgopeed.subscribeTaskEvents(0L, null)
                    } else {
                        Libgopeed.subscribeTaskEvents(
                            mask,
                            object : TaskEventListener {
                                override fun onTaskEvent(payload: String?) {
                                    runOnUiThread {
                                        libgopeedChannel.invokeMethod("taskEvent", payload ?: "")
                                    }
                                }
                            },
                        )
                    }
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
