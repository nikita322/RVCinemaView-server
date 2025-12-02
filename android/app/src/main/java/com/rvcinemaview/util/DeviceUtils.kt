package com.rvcinemaview.util

import android.app.UiModeManager
import android.content.Context
import android.content.res.Configuration

object DeviceUtils {

    fun isTV(context: Context): Boolean {
        val uiModeManager = context.getSystemService(Context.UI_MODE_SERVICE) as UiModeManager
        return uiModeManager.currentModeType == Configuration.UI_MODE_TYPE_TELEVISION
    }

    fun isTouchscreen(context: Context): Boolean {
        return context.packageManager.hasSystemFeature("android.hardware.touchscreen")
    }
}
