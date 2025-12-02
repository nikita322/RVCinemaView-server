package com.rvcinemaview.ui.common

import android.content.Context
import android.content.res.Configuration
import com.rvcinemaview.util.DeviceUtils

/**
 * Configuration for grid column count across different device types.
 * Provides consistent card layout for TV, tablet, and phone.
 */
object GridConfig {

    // Default column counts - can be adjusted as needed
    private const val COLUMNS_PHONE_PORTRAIT = 2
    private const val COLUMNS_PHONE_LANDSCAPE = 4
    private const val COLUMNS_TABLET_PORTRAIT = 3
    private const val COLUMNS_TABLET_LANDSCAPE = 5
    private const val COLUMNS_TV = 6

    // Tablet threshold in dp
    private const val TABLET_MIN_WIDTH_DP = 600

    /**
     * Get the number of columns for the current device and orientation.
     */
    fun getColumnCount(context: Context): Int {
        return when {
            DeviceUtils.isTV(context) -> COLUMNS_TV
            isTablet(context) -> getTabletColumns(context)
            else -> getPhoneColumns(context)
        }
    }

    /**
     * Check if device is a tablet based on screen width.
     */
    private fun isTablet(context: Context): Boolean {
        val widthDp = context.resources.configuration.screenWidthDp
        return widthDp >= TABLET_MIN_WIDTH_DP
    }

    /**
     * Check if device is in landscape orientation.
     */
    private fun isLandscape(context: Context): Boolean {
        return context.resources.configuration.orientation == Configuration.ORIENTATION_LANDSCAPE
    }

    private fun getPhoneColumns(context: Context): Int {
        return if (isLandscape(context)) COLUMNS_PHONE_LANDSCAPE else COLUMNS_PHONE_PORTRAIT
    }

    private fun getTabletColumns(context: Context): Int {
        return if (isLandscape(context)) COLUMNS_TABLET_LANDSCAPE else COLUMNS_TABLET_PORTRAIT
    }

    /**
     * Calculate thumbnail aspect ratio height based on width.
     * Uses 16:9 aspect ratio for video thumbnails.
     */
    fun calculateThumbnailHeight(width: Int): Int {
        return (width * 9) / 16
    }
}
