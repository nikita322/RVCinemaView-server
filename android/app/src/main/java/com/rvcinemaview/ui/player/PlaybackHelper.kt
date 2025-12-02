package com.rvcinemaview.ui.player

import androidx.media3.exoplayer.ExoPlayer
import com.rvcinemaview.data.api.ApiClient
import com.rvcinemaview.data.api.SavePlaybackRequest
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.launch

/**
 * Helper object for common playback operations shared between PlayerActivity and TvPlayerActivity.
 */
object PlaybackHelper {

    /**
     * Load saved position from server and seek player to it.
     * @param scope CoroutineScope for the API call
     * @param mediaId Media ID to load position for
     * @param player ExoPlayer instance to seek
     * @param onPositionLoaded Optional callback with position in seconds
     */
    fun loadSavedPosition(
        scope: CoroutineScope,
        mediaId: String,
        player: ExoPlayer?,
        onPositionLoaded: ((Long) -> Unit)? = null
    ) {
        scope.launch {
            try {
                val response = ApiClient.getApi().getPlaybackPosition(mediaId)
                if (response.position > 0) {
                    player?.seekTo(response.position * 1000) // Convert to milliseconds
                    onPositionLoaded?.invoke(response.position)
                }
            } catch (e: Exception) {
                // Ignore errors - just start from beginning
            }
        }
    }

    /**
     * Save current playback position to server.
     * @param scope CoroutineScope for the API call
     * @param mediaId Media ID to save position for
     * @param player ExoPlayer instance to get position from
     */
    fun savePlaybackPosition(
        scope: CoroutineScope,
        mediaId: String,
        player: ExoPlayer?
    ) {
        val exoPlayer = player ?: return

        val positionMs = exoPlayer.currentPosition
        val durationMs = exoPlayer.duration

        if (durationMs <= 0) return

        val positionSec = positionMs / 1000
        val durationSec = durationMs / 1000

        scope.launch {
            try {
                ApiClient.getApi().savePlaybackPosition(
                    mediaId,
                    SavePlaybackRequest(positionSec, durationSec)
                )
            } catch (e: Exception) {
                // Ignore save errors silently
            }
        }
    }
}
