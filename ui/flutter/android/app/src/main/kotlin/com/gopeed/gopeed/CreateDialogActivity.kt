package com.gopeed.gopeed

import android.os.Bundle
import android.view.Gravity
import android.view.WindowManager
import io.flutter.embedding.android.FlutterActivityLaunchConfigs

class CreateDialogActivity : MainActivity() {
    override fun isDialogMode(): Boolean = true

    override fun getBackgroundMode(): FlutterActivityLaunchConfigs.BackgroundMode {
        return FlutterActivityLaunchConfigs.BackgroundMode.transparent
    }

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)

        val metrics = resources.displayMetrics
        val width = (metrics.widthPixels * 0.92f).toInt()
        val maxHeight = (metrics.heightPixels * 0.62f).toInt()
        val preferredHeight = (420 * metrics.density).toInt()

        window.setGravity(Gravity.CENTER)
        window.setDimAmount(0.28f)
        window.setLayout(width, minOf(maxHeight, preferredHeight))

        val attrs = window.attributes
        attrs.flags = attrs.flags or WindowManager.LayoutParams.FLAG_DIM_BEHIND
        window.attributes = attrs
    }
}
